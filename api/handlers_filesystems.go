package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/YaleSpinup/apierror"
	yefs "github.com/YaleSpinup/efs-api/efs"
	"github.com/aws/aws-sdk-go/aws"
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

	req := FileSystemCreateRequest{}
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		msg := fmt.Sprintf("cannot decode body into create filesystem input: %s", err)
		handleError(w, apierror.New(apierror.ErrBadRequest, msg, err))
		return
	}

	output, task, err := s.filesystemCreate(r.Context(), account, group, &req)
	if err != nil {
		handleError(w, err)
		return
	}

	j, err := json.Marshal(output)
	if err != nil {
		log.Errorf("cannot marshal response (%v) into JSON: %s", output, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("X-Flywheel-Task", task.ID)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
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

// FileSystemShowHandler gets the details if a filesystem by id
func (s *server) FileSystemShowHandler(w http.ResponseWriter, r *http.Request) {
	w = LogWriter{w}
	vars := mux.Vars(r)
	account := s.mapAccountNumber(vars["account"])
	group := vars["group"]
	fs := vars["id"]

	role := fmt.Sprintf("arn:aws:iam::%s:role/%s", account, s.session.RoleName)
	policy, err := generatePolicy("elasticfilesystem:*", "kms:*")
	if err != nil {
		log.Errorf("cannot generate policy for role: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	session, err := s.assumeRole(
		r.Context(),
		s.session.ExternalID,
		role,
		policy,
	)
	if err != nil {
		log.Errorf("cannot assume role: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	efsService := yefs.New(yefs.WithSession(session.Session))

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

	backup, err := efsService.GetFilesystemBackup(r.Context(), aws.StringValue(filesystem.FileSystemId))
	if err != nil {
		handleError(w, err)
		return
	}

	transitionToIA, transitionToPrimary, err := efsService.GetFilesystemLifecycle(r.Context(), aws.StringValue(filesystem.FileSystemId))
	if err != nil {
		handleError(w, err)
		return
	}

	accessPoints, err := efsService.ListAccessPoints(r.Context(), aws.StringValue(filesystem.FileSystemId))
	if err != nil {
		handleError(w, err)
		return
	}

	policyString, err := efsService.GetFileSystemPolicy(r.Context(), aws.StringValue(filesystem.FileSystemId))
	if err != nil {
		if aerr, ok := errors.Cause(err).(apierror.Error); ok {
			if aerr.Code != apierror.ErrNotFound {
				log.Errorf("error: %s", aerr)
				handleError(w, err)
				return
			}
		}
	}

	fsPolicy, err := filSystemAccessPolicyFromEfsPolicy(policyString)
	if err != nil {
		handleError(w, err)
		return
	}

	output := fileSystemResponseFromEFS(filesystem, mounttargets, accessPoints, fsPolicy, backup, transitionToIA, transitionToPrimary)
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

// FileSystemDeleteHandler deletes a filesystem by id
func (s *server) FileSystemDeleteHandler(w http.ResponseWriter, r *http.Request) {
	w = LogWriter{w}
	vars := mux.Vars(r)
	account := vars["account"]
	group := vars["group"]
	fs := vars["id"]

	task, err := s.filesystemDelete(r.Context(), account, group, fs)
	if err != nil {
		handleError(w, err)
		return
	}

	w.Header().Set("X-Flywheel-Task", task.ID)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	w.Write([]byte("OK"))
}

// FileSystemUpdateHandler updates a filesystem by id
func (s *server) FileSystemUpdateHandler(w http.ResponseWriter, r *http.Request) {
	w = LogWriter{w}
	vars := mux.Vars(r)
	account := vars["account"]
	group := vars["group"]
	fs := vars["id"]

	if exists, err := s.fileSystemExists(r.Context(), account, group, fs); err != nil {
		handleError(w, err)
	} else if !exists {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	req := FileSystemUpdateRequest{}
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		msg := fmt.Sprintf("cannot decode body into update filesystem input: %s", err)
		handleError(w, apierror.New(apierror.ErrBadRequest, msg, err))
		return
	}

	task, err := s.filesystemUpdate(r.Context(), account, group, fs, &req)
	if err != nil {
		handleError(w, err)
		return
	}

	w.Header().Set("X-Flywheel-Task", task.ID)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	w.Write([]byte("OK"))
}