package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/YaleSpinup/apierror"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/efs"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

// FileSystemCreateHandler creates a filesystem service
func (s *server) FileSystemCreateHandler(w http.ResponseWriter, r *http.Request) {
	w = LogWriter{w}
	vars := mux.Vars(r)
	account := vars["account"]
	efsService, ok := s.efsServices[account]
	if !ok {
		log.Errorf("account not found: %s", account)
		w.WriteHeader(http.StatusNotFound)
		return
	}

	req := FileSystemCreateRequest{}
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		msg := fmt.Sprintf("cannot decode body into create filesystem input: %s", err)
		handleError(w, apierror.New(apierror.ErrBadRequest, msg, err))
		return
	}

	// setup err var, rollback function list and defer execution, note that we depend on the err variable defined above this
	var rollBackTasks []rollbackFunc
	defer func() {
		if err != nil {
			log.Errorf("recovering from error: %s, executing %d rollback tasks", err, len(rollBackTasks))
			rollBack(&rollBackTasks)
		}
	}()

	// append org tag that will get applied to all resources that tag
	req.Tags = append(req.Tags, &Tag{
		Key:   "spinup:org",
		Value: s.org,
	})

	if req.Name != "" {
		req.Tags = append(req.Tags, &Tag{
			Key:   "Name",
			Value: req.Name,
		})
	}

	// override encryption key if one was passed
	if req.KmsKeyId == "" {
		req.KmsKeyId = efsService.DefaultKmsKeyId
	}

	creationToken, err := uuid.NewRandom()
	if err != nil {
		handleError(w, apierror.New(apierror.ErrInternalError, "failed to generate creation token uuid", err))
		return
	}

	filesystem, err := efsService.CreateFileSystem(r.Context(), &efs.CreateFileSystemInput{
		CreationToken:   aws.String(creationToken.String()),
		Encrypted:       aws.Bool(true),
		KmsKeyId:        aws.String(req.KmsKeyId),
		PerformanceMode: aws.String("generalPurpose"),
		Tags:            toEFSTags(req.Tags),
	})
	if err != nil {
		handleError(w, err)
		return
	}

	fsid := aws.StringValue(filesystem.FileSystemId)

	// wait for the filesystem to become available
	err = retry(3, 2*time.Second, func() error {
		log.Infof("checking if filesystem %s is available before continuing", fsid)

		out, err := efsService.GetFileSystem(r.Context(), fsid)
		if err != nil {
			log.Warnf("got error checking for filesystem is available: %s", err)
			return err
		}

		if status := aws.StringValue(out.LifeCycleState); status != "available" {
			log.Warnf("filesystem %s is not yet available (%s)", fsid, status)
			return fmt.Errorf("filsystem %s not yet available", fsid)
		}

		log.Infof("filesystem %s is available", fsid)
		return nil
	})

	if err != nil {
		msg := fmt.Sprintf("failed to create filesystem %s, timeout waiting to become available: %s", fsid, err.Error())
		handleError(w, errors.Wrap(err, msg))
		return
	}

	// TODO rollback
	mounttargets := []*efs.MountTargetDescription{}
	for _, subnet := range efsService.DefaultSubnets {
		mt, err := efsService.CreateMountTarget(r.Context(), &efs.CreateMountTargetInput{
			FileSystemId: filesystem.FileSystemId,
			// SecurityGroups: TODO,
			SubnetId: aws.String(subnet),
		})

		if err != nil {
			handleError(w, err)
			return
		}

		mounttargets = append(mounttargets, mt)

		// TODO rollback
	}

	output := fileSystemResponseFromEFS(filesystem, mounttargets, nil)
	j, err := json.Marshal(output)
	if err != nil {
		log.Errorf("cannot marshal response (%v) into JSON: %s", output, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(j)
}

// FilesystemListHandler lists all of the filesystems by id
func (s *server) FileSystemListHandler(w http.ResponseWriter, r *http.Request) {
	w = LogWriter{w}
	vars := mux.Vars(r)
	account := vars["account"]
	efsService, ok := s.efsServices[account]
	if !ok {
		log.Errorf("account not found: %s", account)
		w.WriteHeader(http.StatusNotFound)
		return
	}

	out, err := efsService.ListFileSystems(r.Context(), &efs.DescribeFileSystemsInput{})
	if err != nil {
		handleError(w, err)
		return
	}

	output := listFileSystemsResponse(out)
	j, err := json.Marshal(output)
	if err != nil {
		log.Errorf("cannot marshal response (%v) into JSON: %s", output, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(j)
}

// FilesystemShowHandler gets the details if a filesystem by id
func (s *server) FileSystemShowHandler(w http.ResponseWriter, r *http.Request) {
	w = LogWriter{w}
	vars := mux.Vars(r)
	account := vars["account"]
	fs := vars["id"]

	efsService, ok := s.efsServices[account]
	if !ok {
		log.Errorf("account not found: %s", account)
		w.WriteHeader(http.StatusNotFound)
		return
	}

	filesystem, err := efsService.GetFileSystem(r.Context(), fs)
	if err != nil {
		handleError(w, err)
		return
	}

	mounttargets, err := efsService.ListMountTargetsForFileSystem(r.Context(), aws.StringValue(filesystem.FileSystemId))
	if err != nil {
		handleError(w, err)
		return
	}

	output := fileSystemResponseFromEFS(filesystem, mounttargets, nil)
	j, err := json.Marshal(output)
	if err != nil {
		log.Errorf("cannot marshal response (%v) into JSON: %s", output, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(j)
}

// FilesystemDeleteHandler deletes a filesystem by id
func (s *server) FileSystemDeleteHandler(w http.ResponseWriter, r *http.Request) {
	w = LogWriter{w}
	vars := mux.Vars(r)
	account := vars["account"]
	fs := vars["id"]

	efsService, ok := s.efsServices[account]
	if !ok {
		log.Errorf("account not found: %s", account)
		w.WriteHeader(http.StatusNotFound)
		return
	}

	filesystem, err := efsService.GetFileSystem(r.Context(), fs)
	if err != nil {
		handleError(w, err)
		return
	}

	if status := aws.StringValue(filesystem.LifeCycleState); status != "available" {
		msg := fmt.Sprintf("filesystem %s has status %s, cannot delete", fs, status)
		handleError(w, apierror.New(apierror.ErrConflict, msg, nil))
		return
	}

	mounttargets, err := efsService.ListMountTargetsForFileSystem(r.Context(), fs)
	if err != nil {
		handleError(w, err)
		return
	}

	for _, mt := range mounttargets {
		if status := aws.StringValue(filesystem.LifeCycleState); status != "available" {
			msg := fmt.Sprintf("filesystem %s mount target %s has status %s, cannot delete", fs, aws.StringValue(mt.MountTargetId), status)
			handleError(w, apierror.New(apierror.ErrConflict, msg, nil))
			return
		}

		if err := efsService.DeleteMountTarget(r.Context(), aws.StringValue(mt.MountTargetId)); err != nil {
			handleError(w, err)
			return
		}
	}

	err = retry(5, 2*time.Second, func() error {
		out, err := efsService.GetFileSystem(r.Context(), fs)
		if err != nil {
			log.Warnf("error getting filesystem %s during delete: %s", fs, err)
			return err
		}

		if num := aws.Int64Value(out.NumberOfMountTargets); num > 0 {
			log.Warnf("number of mount targets for filesystem %s > 0 (current: %d)", fs, num)
			return fmt.Errorf("waiting for number of mount targets for filesystem %s to be 0 (current: %d)", fs, num)
		}

		return nil
	})
	if err != nil {
		handleError(w, err)
		return
	}

	err = retry(3, 2*time.Second, func() error {
		if err := efsService.DeleteFileSystem(r.Context(), fs); err != nil {
			log.Warnf("error deleting filesystem %s: %s", fs, err)
			return err
		}

		return nil
	})
	if err != nil {
		handleError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}
