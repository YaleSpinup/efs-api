package api

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/YaleSpinup/apierror"
	"github.com/YaleSpinup/flywheel"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/efs"

	yiam "github.com/YaleSpinup/aws-go/services/iam"
	yefs "github.com/YaleSpinup/efs-api/efs"

	log "github.com/sirupsen/logrus"
)

// filesystemCreate orchestrates the creation of an EFS filesystem and all related mount targets, policies, etc.
func (s *server) filesystemCreate(ctx context.Context, account, group string, req *FileSystemCreateRequest) (*FileSystemResponse, *flywheel.Task, error) {
	service, ok := s.efsServices[account]
	if !ok {
		return nil, nil, apierror.New(apierror.ErrNotFound, "account doesnt exist", nil)
	}

	// TODO The following mapping of account numbers hould be updated once the filesystem orchestrations
	// functions are moved to use account numbers (instead of aliases) and to use orchestrators instead
	// of depending on the server struct.
	acctNum := s.mapAccountNumber(account)

	if req.Name == "" {
		return nil, nil, apierror.New(apierror.ErrBadRequest, "Name is a required field", nil)
	}

	// normalize the tags passed in the request
	req.Tags = normalizeTags(s.org, req.Name, group, req.Tags)

	// override encryption key if one was passed
	if req.KmsKeyId == "" {
		req.KmsKeyId = service.DefaultKmsKeyId
	}

	// validate lifecycle configuration setting
	switch req.LifeCycleConfiguration {
	case "", "NONE":
		req.LifeCycleConfiguration = "NONE"
	case "AFTER_7_DAYS",
		"AFTER_14_DAYS",
		"AFTER_30_DAYS",
		"AFTER_60_DAYS",
		"AFTER_90_DAYS":
		log.Debugf("setting Tansition to Infrequent access to %s", req.LifeCycleConfiguration)
	default:
		return nil, nil, apierror.New(apierror.ErrBadRequest, "invalid lifecycle configuration, valid values are NONE | AFTER_7_DAYS | AFTER_14_DAYS | AFTER_30_DAYS | AFTER_60_DAYS | AFTER_90_DAYS", nil)
	}

	// validate intelligent tiering configuration setting
	switch req.TransitionToPrimaryStorageClass {
	case "", "NONE":
		req.TransitionToPrimaryStorageClass = "NONE"
	case "AFTER_1_ACCESS":
		log.Debugf("setting Tansition to primary access to %s", req.TransitionToPrimaryStorageClass)
	default:
		return nil, nil, apierror.New(apierror.ErrBadRequest, "invalid transition to primary storage class rule, valid values are NONE | AFTER_1_ACCESS", nil)
	}

	// validate backup policy setting
	switch req.BackupPolicy {
	case "":
		req.BackupPolicy = "DISABLED"
	case "DISABLED", "ENABLED":
		log.Debugf("setting backup policy to %s", req.BackupPolicy)
	default:
		return nil, nil, apierror.New(apierror.ErrBadRequest, "invalid backup policy, valid values are ENABLED | DISABLED", nil)
	}

	// generate a new task to track and start it
	task := flywheel.NewTask()
	input := efs.CreateFileSystemInput{
		CreationToken:   aws.String(task.ID),
		Encrypted:       aws.Bool(true),
		KmsKeyId:        aws.String(req.KmsKeyId),
		PerformanceMode: aws.String("generalPurpose"),
		Tags:            toEFSTags(req.Tags),
	}

	// if subnets were not passed with the request, set them from the defaults
	if req.Subnets == nil {
		req.Subnets = service.DefaultSubnets
	}

	if req.OneZone {
		var err error
		subnets, err := s.subnetAzs(ctx, account, req.Subnets)
		if err != nil {
			return nil, nil, err
		}

		// get a "random" az/subnet from the map
		var az, subnet string
		for subnet, az = range subnets {
			break
		}

		log.Debugf("setting availability zone to %s", az)

		input.AvailabilityZoneName = aws.String(az)
		req.Subnets = []string{subnet}
	}

	// create the filessystem
	filesystem, err := service.CreateFileSystem(ctx, &input)
	if err != nil {
		return nil, nil, err
	}

	// start the async orchestration to wait for filesystem to become available, set policies and create mount targets
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

		rollBackTasks = append(rollBackTasks, func(ctx context.Context) error {
			log.Errorf("rollback: deleting filesystem: %s", fsid)
			return service.DeleteFileSystem(fsCtx, fsid)
		})

		msgChan <- fmt.Sprintf("setting filesystem %s backup policy to %s", fsid, req.BackupPolicy)

		err = service.SetFileSystemBackup(fsCtx, fsid, req.BackupPolicy)
		if err != nil {
			errChan <- fmt.Errorf("failed to set backup policy for filesystem %s: %s", fsid, err.Error())
			return
		}
		msgChan <- fmt.Sprintf("setting filesystem %s lifecycle configuration to %s", fsid, req.LifeCycleConfiguration)

		err = service.SetFileSystemLifecycle(fsCtx, fsid, req.LifeCycleConfiguration, req.TransitionToPrimaryStorageClass)
		if err != nil {
			errChan <- fmt.Errorf("failed to set lifecycle for filesystem %s: %s", fsid, err.Error())
			return
		}

		if req.AccessPolicy != nil {
			msgChan <- fmt.Sprintf("setting filesystem %s access policy to %+v", fsid, req.AccessPolicy)

			var policy []byte
			policy, err = json.Marshal(efsPolicyFromFilSystemAccessPolicy(acctNum, group, aws.StringValue(filesystem.FileSystemArn), req.AccessPolicy))
			if err != nil {
				errChan <- fmt.Errorf("failed to marshall access policy for filesystem %s: %s", fsid, err.Error())
				return
			}

			err = service.SetFileSystemPolicy(fsCtx, fsid, string(policy))
			if err != nil {
				errChan <- fmt.Errorf("failed to set access policy for filesystem %s: %s", fsid, err.Error())
				return
			}
		}

		mounttargets := []*efs.MountTargetDescription{}
		for _, subnet := range req.Subnets {
			if req.Sgs == nil {
				log.Debugf("setting default security groups on mount target")
				req.Sgs = service.DefaultSgs
			}

			var mt *efs.MountTargetDescription
			mt, err = service.CreateMountTarget(fsCtx, &efs.CreateMountTargetInput{
				FileSystemId:   aws.String(fsid),
				SecurityGroups: aws.StringSlice(req.Sgs),
				SubnetId:       aws.String(subnet),
			})

			if err != nil {
				errChan <- fmt.Errorf("failed to create mount target for filesystem %s: %s", fsid, err)
				return
			}

			mounttargets = append(mounttargets, mt)

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

		rollBackTasks = append(rollBackTasks, func(ctx context.Context) error {
			log.Errorf("rollback: deleting mount target for filesystem %s", fsid)

			for _, mt := range mounttargets {
				if err := service.DeleteMountTarget(fsCtx, aws.StringValue(mt.MountTargetId)); err != nil {
					return err
				}
			}

			if err = retry(10, 2*time.Second, func() error {
				log.Warnf("rollback: waiting for number of mount targets for filesystem %s to be 0", fsid)

				out, err := service.GetFileSystem(fsCtx, fsid)
				if err != nil {
					log.Warnf("rollback: error getting filesystem %s during delete: %s", fsid, err)
					return err
				}

				if num := aws.Int64Value(out.NumberOfMountTargets); num > 0 {
					log.Warnf("number of mount targets for filesystem %s > 0 (current: %d)", fsid, num)
					return fmt.Errorf("waiting for number of mount targets for filesystem %s to be 0 (current: %d)", fsid, num)
				}

				return nil
			}); err != nil {
				return err
			}

			return nil
		})

		for _, apReq := range req.AccessPoints {
			msgChan <- fmt.Sprintf("creating access point '%s' for fs %s", apReq.Name, fsid)

			var ap *AccessPoint
			var apTask *flywheel.Task
			ap, apTask, err = s.accessPointCreate(fsCtx, account, group, fsid, apReq)
			if err != nil {
				errChan <- err
				return
			}

			if err = retry(10, 2*time.Second, func() error {
				msgChan <- fmt.Sprintf("waiting for access point %s for filesystem %s to be available", ap.AccessPointId, fsid)

				apt, err := s.flywheel.GetTask(fsCtx, apTask.ID)
				if err != nil {
					return err
				}

				if apt == nil {
					return fmt.Errorf("filsystem %s not found", fsid)
				}

				switch apt.Status {
				case flywheel.STATUS_COMPLETED:
					return nil
				case flywheel.STATUS_FAILED:
					return fmt.Errorf("failed to create access point %s for fs %s: %s", ap.AccessPointId, fsid, apTask.Failure)
				}

				msgChan <- fmt.Sprintf("access point %s for filesystem %s is not yet available (%s)", ap.AccessPointId, fsid, apTask.Status)
				return fmt.Errorf("filsystem %s not yet available", fsid)
			}); err != nil {
				errChan <- err
				return
			}
		}

	}()

	return fileSystemResponseFromEFS(filesystem, nil, nil, req.AccessPolicy, req.BackupPolicy, req.LifeCycleConfiguration, req.TransitionToPrimaryStorageClass), task, nil
}

func (s *server) filesystemUpdate(ctx context.Context, account, group, fs string, req *FileSystemUpdateRequest) (*flywheel.Task, error) {
	// TODO The following mapping of account numbers hould be updated once the filesystem orchestrations
	// functions are moved to use account numbers (instead of aliases) and to use orchestrators instead
	// of depending on the server struct.
	acctNum := s.mapAccountNumber(account)

	service, ok := s.efsServices[account]
	if !ok {
		return nil, apierror.New(apierror.ErrNotFound, "account doesnt exist", nil)
	}

	filesystem, err := service.GetFileSystem(ctx, fs)
	if err != nil {
		return nil, err
	}

	// if the lifecycle configuraiton or transition to primary storage rule is updated
	if req.LifeCycleConfiguration != "" || req.TransitionToPrimaryStorageClass != "" {
		transitionToIA, transitionToPrimary, err := service.GetFilesystemLifecycle(ctx, fs)
		if err != nil {
			return nil, err
		}

		switch req.LifeCycleConfiguration {
		case "":
			log.Debugf("not updating lifecycle configuration")
			req.LifeCycleConfiguration = transitionToIA
		case "NONE":
			req.LifeCycleConfiguration = "NONE"
		case "AFTER_7_DAYS",
			"AFTER_14_DAYS",
			"AFTER_30_DAYS",
			"AFTER_60_DAYS",
			"AFTER_90_DAYS":
			log.Debugf("setting Transition to Infrequent access to %s", req.LifeCycleConfiguration)
		default:
			return nil, apierror.New(apierror.ErrBadRequest, "invalid lifecycle configuration, valid values are NONE | AFTER_7_DAYS | AFTER_14_DAYS | AFTER_30_DAYS | AFTER_60_DAYS | AFTER_90_DAYS", nil)
		}

		// validate intelligent tiering configuration setting
		switch req.TransitionToPrimaryStorageClass {
		case "":
			log.Debugf("not updating intelligent tiering rule")
			req.TransitionToPrimaryStorageClass = transitionToPrimary
		case "NONE":
			req.TransitionToPrimaryStorageClass = "NONE"
		case "AFTER_1_ACCESS":
			log.Debugf("setting Tansition to primary access to %s", req.TransitionToPrimaryStorageClass)
		default:
			return nil, apierror.New(apierror.ErrBadRequest, "invalid transition to primary storage class rule, valid values are NONE | AFTER_1_ACCESS", nil)
		}

	}

	switch req.BackupPolicy {
	case "":
		log.Debugf("not updatingbackup policy")
	case "DISABLED", "ENABLED":
		log.Debugf("setting backup policy to %s", req.BackupPolicy)
	default:
		return nil, apierror.New(apierror.ErrBadRequest, "invalid backup policy, valid values are ENABLED | DISABLED", nil)
	}

	if req.Tags != nil {
		// normalize the tags passed in the request
		req.Tags = normalizeTags(s.org, aws.StringValue(filesystem.Name), group, req.Tags)
	}

	// generate a new task to track and start it
	task := flywheel.NewTask()

	// start the orchestration
	go func() {
		fsid := aws.StringValue(filesystem.FileSystemId)

		fsCtx, cancel := context.WithCancel(context.Background())
		defer cancel()

		msgChan, errChan := s.startTask(fsCtx, task)

		msgChan <- fmt.Sprintf("requested update of filesystem %s", fsid)

		if req.BackupPolicy != "" {
			msgChan <- fmt.Sprintf("setting filesystem %s backup policy to %s", fsid, req.BackupPolicy)

			if err := service.SetFileSystemBackup(fsCtx, fsid, req.BackupPolicy); err != nil {
				errChan <- fmt.Errorf("failed to set backup policy for filesystem %s: %s", fsid, err.Error())
				return
			}
		}

		if req.LifeCycleConfiguration != "" || req.TransitionToPrimaryStorageClass != "" {
			msgChan <- fmt.Sprintf("setting filesystem %s lifecycle configuration to %s", fsid, req.LifeCycleConfiguration)

			if err := service.SetFileSystemLifecycle(fsCtx, fsid, req.LifeCycleConfiguration, req.TransitionToPrimaryStorageClass); err != nil {
				errChan <- fmt.Errorf("failed to set lifecycle for filesystem %s: %s", fsid, err.Error())
				return
			}
		}

		if req.AccessPolicy != nil {
			msgChan <- fmt.Sprintf("setting filesystem %s access policy to %+v", fsid, req.AccessPolicy)

			var policy []byte
			policy, err = json.Marshal(efsPolicyFromFilSystemAccessPolicy(acctNum, group, aws.StringValue(filesystem.FileSystemArn), req.AccessPolicy))
			if err != nil {
				errChan <- fmt.Errorf("failed to marshall access policy for filesystem %s: %s", fsid, err.Error())
				return
			}

			err = service.SetFileSystemPolicy(fsCtx, fsid, string(policy))
			if err != nil {
				errChan <- fmt.Errorf("failed to set access policy for filesystem %s: %s", fsid, err.Error())
				return
			}
		}

		if req.Tags != nil {
			msgChan <- fmt.Sprintf("updating tags for filesystem %s ", fsid)

			if err := service.TagFilesystem(fsCtx, fsid, toEFSTags(req.Tags)); err != nil {
				errChan <- fmt.Errorf("failed to set tags for filesystem %s: %s", fsid, err.Error())
				return
			}

			// TODO The following assumerole code should be updated
			// once the filesystem orchestrations functions are moved to use account numbers (instead of
			// aliases) and to use orchestrators instead of depending on the server struct.
			role := fmt.Sprintf("arn:aws:iam::%s:role/%s", acctNum, s.session.RoleName)

			// IAM doesn't support resource tags, so we can't pass the s.orgPolicy here
			session, err := s.assumeRole(
				fsCtx,
				s.session.ExternalID,
				role,
				"",
				"arn:aws:iam::aws:policy/IAMReadOnlyAccess",
				"arn:aws:iam::aws:policy/AmazonElasticFileSystemReadOnlyAccess",
			)
			if err != nil {
				return
			}

			efsService := yefs.New(yefs.WithSession(session.Session))
			iamService := yiam.New(yiam.WithSession(session.Session))

			orch := newUserOrchestrator(iamService, efsService, s.org)

			users, err := orch.listFilesystemUsers(fsCtx, acctNum, group, fsid)
			if err != nil {
				errChan <- fmt.Errorf("failed to list users filesystem %s: %s", fsid, err.Error())
				return
			}

			for _, u := range users {
				if err := s.updateTagsForUser(fsCtx, acctNum, group, fsid, u, req.Tags); err != nil {
					errChan <- fmt.Errorf("failed to update tags for users of filesystem %s: %s", fsid, err.Error())
					return
				}
			}
		}

	}()

	return task, nil
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

	accesspoints, err := service.ListAccessPoints(ctx, fs)
	if err != nil {
		return nil, err
	}

	// if there are any accesspoints defined for the filesystem, delete them
	for _, ap := range accesspoints {
		if err := service.DeleteAccessPoint(ctx, aws.StringValue(ap.AccessPointId)); err != nil {
			return nil, err
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

		// TODO The following mapping of account numbers and assumerole code should be updated
		// once the filesystem orchestrations functions are moved to use account numbers (instead of
		// aliases) and to use orchestrators instead of depending on the server struct.
		acctNum := s.mapAccountNumber(account)

		role := fmt.Sprintf("arn:aws:iam::%s:role/%s", acctNum, s.session.RoleName)
		policy, err := s.filesystemUserDeletePolicy()
		if err != nil {
			errChan <- err
			return
		}

		// IAM doesn't support resource tags, so we can't pass the s.orgPolicy here
		session, err := s.assumeRole(
			fsCtx,
			s.session.ExternalID,
			role,
			policy,
			"arn:aws:iam::aws:policy/AmazonElasticFileSystemReadOnlyAccess",
		)
		if err != nil {
			errChan <- err
			return
		}

		efsService := yefs.New(yefs.WithSession(session.Session))
		iamService := yiam.New(yiam.WithSession(session.Session))

		orch := newUserOrchestrator(iamService, efsService, s.org)

		users, err := orch.deleteAllFilesystemUsers(fsCtx, acctNum, group, fsid)
		if err != nil {
			errChan <- err
			return
		}

		msgChan <- fmt.Sprintf("deleted filesystem %s users %+v", fsid, users)

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

// subnetAzs returns a map of subnets to availability zone names used by EFS
// this may change to support getting the list of subnets as well, currently it uses
// the defaults from the EFS service
func (s *server) subnetAzs(ctx context.Context, account string, defSubnets []string) (map[string]string, error) {
	log.Infof("determining availability zone for account %s and subnets %+v", account, defSubnets)

	ec2Service, ok := s.ec2Services[account]
	if !ok {
		return nil, apierror.New(apierror.ErrNotFound, "account doesnt exist", nil)
	}

	subnets := make(map[string]string)
	for _, s := range defSubnets {
		subnet, err := ec2Service.GetSubnet(ctx, s)
		if err != nil {
			return nil, err
		}

		log.Debugf("got details about subnet %s: %+v", s, subnet)

		subnets[s] = aws.StringValue(subnet.AvailabilityZone)
	}

	if len(subnets) == 0 {
		return nil, apierror.New(apierror.ErrBadRequest, "failed to determine usable availability zone", nil)
	}

	return subnets, nil
}
