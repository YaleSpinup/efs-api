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
	group := vars["group"]
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

	// normalize the tags passed in the request
	req.Tags = s.normalizeTags(req.Name, group, req.Tags)

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
		if req.Sgs == nil {
			log.Debugf("setting default security groups on mount target")
			req.Sgs = efsService.DefaultSgs
		}

		mt, err := efsService.CreateMountTarget(r.Context(), &efs.CreateMountTargetInput{
			FileSystemId:   filesystem.FileSystemId,
			SecurityGroups: aws.StringSlice(req.Sgs),
			SubnetId:       aws.String(subnet),
		})

		if err != nil {
			handleError(w, err)
			return
		}

		mounttargets = append(mounttargets, mt)

		// TODO rollback
		// TODO tag mount target eni?
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

// FileSystemListHandler lists all of the filesystems in a group by id
func (s *server) FileSystemListHandler(w http.ResponseWriter, r *http.Request) {
	w = LogWriter{w}
	vars := mux.Vars(r)
	account := vars["account"]
	group := vars["group"]

	out, err := s.listFileSystems(r.Context(), account, group)
	if err != nil {
		handleError(w, err)
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
	group := vars["group"]
	fs := vars["id"]

	efsService, ok := s.efsServices[account]
	if !ok {
		log.Errorf("account not found: %s", account)
		w.WriteHeader(http.StatusNotFound)
		return
	}

	if exists, err := s.fileSystemExists(r.Context(), account, group, fs); err != nil {
		handleError(w, err)
	} else if !exists {
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
	group := vars["group"]
	fs := vars["id"]

	efsService, ok := s.efsServices[account]
	if !ok {
		log.Errorf("account not found: %s", account)
		w.WriteHeader(http.StatusNotFound)
		return
	}

	if exists, err := s.fileSystemExists(r.Context(), account, group, fs); err != nil {
		handleError(w, err)
	} else if !exists {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	filesystem, err := efsService.GetFileSystem(r.Context(), fs)
	if err != nil {
		handleError(w, err)
		return
	}

	if status := aws.StringValue(filesystem.LifeCycleState); status != "available" {
		msg := fmt.Sprintf("filesystem %s has status %s, cannot delete filesystems that are not 'available'", fs, status)
		handleError(w, apierror.New(apierror.ErrConflict, msg, nil))
		return
	}

	mounttargets, err := efsService.ListMountTargetsForFileSystem(r.Context(), fs)
	if err != nil {
		handleError(w, err)
		return
	}

	for _, mt := range mounttargets {
		if status := aws.StringValue(mt.LifeCycleState); status != "available" {
			msg := fmt.Sprintf("filesystem %s mount target %s has status %s, cannot delete", fs, aws.StringValue(mt.MountTargetId), status)
			handleError(w, apierror.New(apierror.ErrConflict, msg, nil))
			return
		}
	}

	go func() {
		fsid := aws.StringValue(filesystem.FileSystemId)
		log.Infof("starting background deletion of filesystem id: %s", fsid)
		if err := efsService.DeleteFileSystemO(fsid); err != nil {
			log.Errorf("Failed to delete filesystem id %s: %s", fsid, err)
		}
	}()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	w.Write([]byte("OK"))
}
