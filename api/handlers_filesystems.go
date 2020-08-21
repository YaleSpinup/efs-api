package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/YaleSpinup/apierror"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
)

// FileSystemCreateHandler creates a filesystem service
func (s *server) FileSystemCreateHandler(w http.ResponseWriter, r *http.Request) {
	w = LogWriter{w}
	vars := mux.Vars(r)
	account := vars["account"]
	group := vars["group"]
	_, ok := s.efsServices[account]
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

	output, task, err := s.filesystemCreate(r.Context(), account, group, &req)
	if err != nil {
		handleError(w, err)
	}

	j, err := json.Marshal(output)
	if err != nil {
		log.Errorf("cannot marshal response (%v) into JSON: %s", output, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("X-Spinup-Task", task.ID)
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

	_, ok := s.efsServices[account]
	if !ok {
		log.Errorf("account not found: %s", account)
		w.WriteHeader(http.StatusNotFound)
		return
	}

	task, err := s.filesystemDelete(r.Context(), account, group, fs)
	if err != nil {
		handleError(w, err)
		return
	}

	w.Header().Set("X-Spinup-Task", task.ID)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	w.Write([]byte("OK"))
}
