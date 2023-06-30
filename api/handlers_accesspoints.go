package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/YaleSpinup/apierror"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
)

// FileSystemAPCreateHandler Route handler for creating the filesystems access points
func (s *server) FileSystemAPCreateHandler(w http.ResponseWriter, r *http.Request) {
	w = LogWriter{w}
	vars := mux.Vars(r)
	account := s.mapAccountNumber(vars["account"])
	fsid := vars["id"]

	req := AccessPointCreateRequest{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		msg := fmt.Sprintf("cannot decode body into create access point request input: %s", err)
		handleError(w, apierror.New(apierror.ErrBadRequest, msg, err))
		return
	}

	output, task, err := s.accessPointCreate(r.Context(), account, fsid, &req)
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

	_, err = w.Write(j)
	if err != nil {
		handleError(w, apierror.New(apierror.ErrInternalError, "error writing response", err))
	}
}

// FileSystemAPListHandler Route handler for listing the filesystems access points
func (s *server) FileSystemAPListHandler(w http.ResponseWriter, r *http.Request) {
	w = LogWriter{w}
	vars := mux.Vars(r)
	account := s.mapAccountNumber(vars["account"])
	fsid := vars["id"]

	out, err := s.listFilesystemAccessPoints(r.Context(), account, fsid)
	if err != nil {
		handleError(w, err)
		return
	}

	j, err := json.Marshal(out)
	if err != nil {
		log.Errorf("cannot marshal response (%v) into JSON: %s", out, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	_, err = w.Write(j)
	if err != nil {
		handleError(w, apierror.New(apierror.ErrInternalError, "error writing response", err))
	}
}

// FileSystemAPShowHandler Request handler for
func (s *server) FileSystemAPShowHandler(w http.ResponseWriter, r *http.Request) {
	w = LogWriter{w}
	vars := mux.Vars(r)
	account := s.mapAccountNumber(vars["account"])
	apid := vars["apid"]

	out, err := s.getFilesystemAccessPoint(r.Context(), account, apid)
	if err != nil {
		handleError(w, err)
		return
	}

	j, err := json.Marshal(out)
	if err != nil {
		log.Errorf("cannot marshal response (%v) into JSON: %s", out, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	_, err = w.Write(j)
	if err != nil {
		handleError(w, apierror.New(apierror.ErrInternalError, "error writing response", err))
	}
}

// FileSystemAPDeleteHandler Request handler for deleting a file system access point
func (s *server) FileSystemAPDeleteHandler(w http.ResponseWriter, r *http.Request) {
	w = LogWriter{w}
	vars := mux.Vars(r)
	account := s.mapAccountNumber(vars["account"])
	apid := vars["apid"]

	if err := s.deleteFilesystemAccessPoint(r.Context(), account, apid); err != nil {
		handleError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	_, err := w.Write([]byte("OK"))
	if err != nil {
		handleError(w, apierror.New(apierror.ErrInternalError, "error writing response", err))
	}
}
