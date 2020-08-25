package api

import (
	"context"
	"fmt"
	"time"

	"github.com/YaleSpinup/apierror"
	"github.com/YaleSpinup/flywheel"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/efs"
	log "github.com/sirupsen/logrus"
)

func (s *server) filesystemCreate(ctx context.Context, account, group string, req *FileSystemCreateRequest) (*FileSystemResponse, *flywheel.Task, error) {
	service, ok := s.efsServices[account]
	if !ok {
		return nil, nil, apierror.New(apierror.ErrNotFound, "account doesnt exist", nil)
	}

	// normalize the tags passed in the request
	req.Tags = s.normalizeTags(req.Name, group, req.Tags)

	// override encryption key if one was passed
	if req.KmsKeyId == "" {
		req.KmsKeyId = service.DefaultKmsKeyId
	}

	// generate a new task to track and start it
	task := flywheel.NewTask()
	filesystem, err := service.CreateFileSystem(ctx, &efs.CreateFileSystemInput{
		CreationToken:   aws.String(task.ID),
		Encrypted:       aws.Bool(true),
		KmsKeyId:        aws.String(req.KmsKeyId),
		PerformanceMode: aws.String("generalPurpose"),
		Tags:            toEFSTags(req.Tags),
	})
	if err != nil {
		return nil, nil, err
	}

	// start the orchestration
	go func() {
		fsid := aws.StringValue(filesystem.FileSystemId)

		fsCtx, cancel := context.WithCancel(context.Background())
		defer cancel()

		msgChan, errChan := s.startTask(fsCtx, task)

		msgChan <- fmt.Sprintf("requested creation of filesystem %s", fsid)

		// setup err var, rollback function list and defer execution
		var err error
		var rollBackTasks []rollbackFunc
		defer func() {
			if err != nil {
				log.Errorf("recovering from error: %s, executing %d rollback tasks", err, len(rollBackTasks))
				rollBack(&rollBackTasks)
			}
		}()

		// wait for the filesystem to become available
		if err = retry(10, 2*time.Second, func() error {
			msg := fmt.Sprintf("checking if filesystem %s is available before continuing", fsid)
			msgChan <- msg

			out, err := service.GetFileSystem(fsCtx, fsid)
			if err != nil {
				msgChan <- fmt.Sprintf("got error checking if filesystem %s is available: %s", fsid, err)
				return err
			}

			if status := aws.StringValue(out.LifeCycleState); status != "available" {
				msgChan <- fmt.Sprintf("filesystem %s is not yet available (%s)", fsid, status)
				return fmt.Errorf("filsystem %s not yet available", fsid)
			}

			msgChan <- fmt.Sprintf("filesystem %s is available", fsid)
			return nil
		}); err != nil {
			errChan <- fmt.Errorf("failed to create filesystem %s, timeout waiting to become available: %s", fsid, err.Error())
			return
		}

		// TODO rollback

		mounttargets := []*efs.MountTargetDescription{}
		for _, subnet := range service.DefaultSubnets {
			if req.Sgs == nil {
				log.Debugf("setting default security groups on mount target")
				req.Sgs = service.DefaultSgs
			}

			mt, err := service.CreateMountTarget(fsCtx, &efs.CreateMountTargetInput{
				FileSystemId:   aws.String(fsid),
				SecurityGroups: aws.StringSlice(req.Sgs),
				SubnetId:       aws.String(subnet),
			})

			if err != nil {
				errChan <- fmt.Errorf("failed to create mount target for filesystem %s: %s", fsid, err)
				return
			}

			mounttargets = append(mounttargets, mt)

			// TODO rollback
			// TODO tag mount target eni?
		}

		// wait for mount targets to become available
		if err = retry(10, 2*time.Second, func() error {
			msgChan <- fmt.Sprintf("waiting for mount targets for filesystem %s to be available", fsid)

			mounttargets, err := service.ListMountTargetsForFileSystem(fsCtx, fsid)
			if err != nil {
				return fmt.Errorf("failed to list mount targets for filesystem %s: %s", fsid, err)
			}

			for _, mt := range mounttargets {
				if status := aws.StringValue(mt.LifeCycleState); status != "available" {
					msg := fmt.Sprintf("filesystem %s mount target %s has status %s, not available", fsid, aws.StringValue(mt.MountTargetId), status)
					return apierror.New(apierror.ErrNotFound, msg, nil)
				}
			}

			return nil
		}); err != nil {
			errChan <- err
			return
		}

		msgChan <- fmt.Sprintf("created %d mount targets for fs %s", len(mounttargets), fsid)
	}()

	return fileSystemResponseFromEFS(filesystem, nil, nil), task, nil
}

func (s *server) filesystemDelete(ctx context.Context, account, group, fs string) (*flywheel.Task, error) {
	service, ok := s.efsServices[account]
	if !ok {
		return nil, apierror.New(apierror.ErrNotFound, "account doesnt exist", nil)
	}

	if exists, err := s.fileSystemExists(ctx, account, group, fs); err != nil {
		return nil, apierror.New(apierror.ErrBadRequest, "", err)
	} else if !exists {
		return nil, apierror.New(apierror.ErrNotFound, "filesystem doesnt exist", err)
	}

	filesystem, err := service.GetFileSystem(ctx, fs)
	if err != nil {
		return nil, err
	}

	if status := aws.StringValue(filesystem.LifeCycleState); status != "available" {
		msg := fmt.Sprintf("filesystem %s has status %s, cannot delete filesystems that are not 'available'", fs, status)
		return nil, apierror.New(apierror.ErrConflict, msg, nil)
	}

	mounttargets, err := service.ListMountTargetsForFileSystem(ctx, fs)
	if err != nil {
		return nil, err
	}

	for _, mt := range mounttargets {
		if status := aws.StringValue(mt.LifeCycleState); status != "available" {
			msg := fmt.Sprintf("filesystem %s mount target %s has status %s, cannot delete", fs, aws.StringValue(mt.MountTargetId), status)
			return nil, apierror.New(apierror.ErrConflict, msg, nil)
		}
	}

	// generate a new task to track and start it
	task := flywheel.NewTask()
	go func() {
		fsid := aws.StringValue(filesystem.FileSystemId)

		fsCtx, cancel := context.WithCancel(context.Background())
		defer cancel()

		msgChan, errChan := s.startTask(fsCtx, task)

		var err error
		mounttargets, err := service.ListMountTargetsForFileSystem(fsCtx, fsid)
		if err != nil {
			errChan <- err
			return
		}

		msgChan <- fmt.Sprintf("listed mount targets for filesystem %s", fsid)

		for _, mt := range mounttargets {
			if status := aws.StringValue(mt.LifeCycleState); status != "available" {
				errChan <- fmt.Errorf("filesystem %s mount target %s has status %s, cannot delete", fsid, aws.StringValue(mt.MountTargetId), status)
				return
			}

			msgChan <- fmt.Sprintf("mount target %s for filesystem %s is available", mt, fsid)

			if err := service.DeleteMountTarget(fsCtx, aws.StringValue(mt.MountTargetId)); err != nil {
				errChan <- err
				return
			}

			msgChan <- fmt.Sprintf("requested delete for mount target %s for filesystem %s", mt, fsid)
		}

		if err = retry(10, 2*time.Second, func() error {
			msgChan <- fmt.Sprintf("waiting for number of mount targets for filesystem %s to be 0", fsid)

			out, err := service.GetFileSystem(fsCtx, fsid)
			if err != nil {
				log.Warnf("error getting filesystem %s during delete: %s", fsid, err)
				return err
			}

			if num := aws.Int64Value(out.NumberOfMountTargets); num > 0 {
				log.Warnf("number of mount targets for filesystem %s > 0 (current: %d)", fsid, num)
				return fmt.Errorf("waiting for number of mount targets for filesystem %s to be 0 (current: %d)", fsid, num)
			}

			return nil
		}); err != nil {
			errChan <- err
			return
		}

		if err = retry(3, 2*time.Second, func() error {
			msgChan <- fmt.Sprintf("deleting filesystem %s", fsid)

			if err := service.DeleteFileSystem(fsCtx, fsid); err != nil {
				log.Warnf("error deleting filesystem %s: %s", fsid, err)
				return err
			}

			return nil
		}); err != nil {
			errChan <- err
			return
		}
	}()

	return task, nil
}

// startTask starts the flywheel task and receives messages on the channels.  in the future, this
// functionality might be part of the flywheel library
func (s *server) startTask(ctx context.Context, task *flywheel.Task) (chan<- string, chan<- error) {
	msgChan := make(chan string)
	errChan := make(chan error)

	// track the task
	go func() {
		taskCtx, cancel := context.WithCancel(context.Background())
		defer cancel()

		if err := s.flywheel.Start(taskCtx, task); err != nil {
			log.Errorf("failed to start flywheel task, won't be tracked: %s", err)
		}

		for {
			select {
			case msg := <-msgChan:
				log.Info(msg)

				if ferr := s.flywheel.CheckIn(taskCtx, task.ID); ferr != nil {
					log.Errorf("failed to checkin task %s: %s", task.ID, ferr)
				}

				if ferr := s.flywheel.Log(taskCtx, task.ID, msg); ferr != nil {
					log.Errorf("failed to log flywheel message for %s: %s", task.ID, ferr)
				}
			case err := <-errChan:
				log.Error(err)

				if ferr := s.flywheel.Fail(taskCtx, task.ID, err.Error()); ferr != nil {
					log.Errorf("failed to fail flywheel task %s: %s", task.ID, ferr)
				}

				return
			case <-ctx.Done():
				log.Infof("marking task %s complete", task.ID)

				if ferr := s.flywheel.Complete(taskCtx, task.ID); ferr != nil {
					log.Errorf("failed to complete flywheel task %s: %s", task.ID, ferr)
				}

				return
			}
		}
	}()

	return msgChan, errChan
}
