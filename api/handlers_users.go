package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/YaleSpinup/apierror"
	"github.com/YaleSpinup/aws-go/services/iam"
	"github.com/YaleSpinup/efs-api/efs"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
)

// UsersCreateHandler handles user creation requests
func (s *server) UsersCreateHandler(w http.ResponseWriter, r *http.Request) {
	w = LogWriter{w}
	vars := mux.Vars(r)
	account := s.mapAccountNumber(vars["account"])
	group := vars["group"]
	fsid := vars["id"]

	req := FileSystemUserCreateRequest{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		msg := fmt.Sprintf("cannot decode body into create user input: %s", err)
		handleError(w, apierror.New(apierror.ErrBadRequest, msg, err))
		return
	}

	if req.UserName == "" {
		handleError(w, apierror.New(apierror.ErrBadRequest, "Username is a required field", nil))
		return
	}

	log.Infof("creating filesystem %s user %s", fsid, req.UserName)

	role := fmt.Sprintf("arn:aws:iam::%s:role/%s", account, s.session.RoleName)

	policy, err := s.filesystemUserCreatePolicy()
	if err != nil {
		handleError(w, apierror.New(apierror.ErrInternalError, "failed to generate policy", err))
		return
	}

	// IAM doesn't support resource tags, so we can't pass the s.orgPolicy here
	session, err := s.assumeRole(
		r.Context(),
		s.session.ExternalID,
		role,
		policy,
		"arn:aws:iam::aws:policy/AmazonElasticFileSystemReadOnlyAccess",
	)
	if err != nil {
		msg := fmt.Sprintf("failed to assume role in account: %s", account)
		handleError(w, apierror.New(apierror.ErrForbidden, msg, nil))
		return
	}

	efsService := efs.New(efs.WithSession(session.Session))
	iamService := iam.New(iam.WithSession(session.Session))

	orch := newUserOrchestrator(iamService, efsService, s.org)

	if err := orch.prepareAccount(r.Context()); err != nil {
		handleError(w, err)
		return
	}

	out, err := orch.createFilesystemUser(r.Context(), account, group, fsid, &req)
	if err != nil {
		handleError(w, err)
		return
	}

	j, err := json.Marshal(out)
	if err != nil {
		log.Errorf("cannot marshal reasponse(%v) into JSON: %s", out, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(j)
}

// UsersDeleteHandler handles user deletion requests
func (s *server) UsersDeleteHandler(w http.ResponseWriter, r *http.Request) {
	w = LogWriter{w}
	vars := mux.Vars(r)
	account := s.mapAccountNumber(vars["account"])
	group := vars["group"]
	fsid := vars["id"]
	user := vars["user"]

	role := fmt.Sprintf("arn:aws:iam::%s:role/%s", account, s.session.RoleName)

	policy, err := s.filesystemUserDeletePolicy()
	if err != nil {
		handleError(w, apierror.New(apierror.ErrInternalError, "failed to generate policy", err))
		return
	}

	// IAM doesn't support resource tags, so we can't pass the s.orgPolicy here
	session, err := s.assumeRole(
		r.Context(),
		s.session.ExternalID,
		role,
		policy,
		"arn:aws:iam::aws:policy/AmazonElasticFileSystemReadOnlyAccess",
	)
	if err != nil {
		msg := fmt.Sprintf("failed to assume role in account: %s", account)
		handleError(w, apierror.New(apierror.ErrForbidden, msg, nil))
		return
	}

	efsService := efs.New(efs.WithSession(session.Session))
	iamService := iam.New(iam.WithSession(session.Session))

	orch := newUserOrchestrator(iamService, efsService, s.org)

	if err := orch.deleteFilesystemUser(r.Context(), account, group, fsid, user); err != nil {
		handleError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

// UsersListHandler handles requests to list users in a space
func (s *server) UsersListHandler(w http.ResponseWriter, r *http.Request) {
	w = LogWriter{w}
	vars := mux.Vars(r)
	account := s.mapAccountNumber(vars["account"])
	group := vars["group"]
	fsid := vars["id"]

	role := fmt.Sprintf("arn:aws:iam::%s:role/%s", account, s.session.RoleName)

	// IAM doesn't support resource tags, so we can't pass the s.orgPolicy here
	session, err := s.assumeRole(
		r.Context(),
		s.session.ExternalID,
		role,
		"",
		"arn:aws:iam::aws:policy/IAMReadOnlyAccess",
		"arn:aws:iam::aws:policy/AmazonElasticFileSystemReadOnlyAccess",
	)
	if err != nil {
		msg := fmt.Sprintf("failed to assume role in account: %s", account)
		handleError(w, apierror.New(apierror.ErrForbidden, msg, nil))
		return
	}

	efsService := efs.New(efs.WithSession(session.Session))
	iamService := iam.New(iam.WithSession(session.Session))

	orch := newUserOrchestrator(iamService, efsService, s.org)

	out, err := orch.listFilesystemUsers(r.Context(), account, group, fsid)
	if err != nil {
		handleError(w, err)
		return
	}

	j, err := json.Marshal(out)
	if err != nil {
		log.Errorf("cannot marshal reasponse(%v) into JSON: %s", out, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(j)
}

// UsersShowHandler handles requests to show a user
func (s *server) UsersShowHandler(w http.ResponseWriter, r *http.Request) {
	w = LogWriter{w}
	vars := mux.Vars(r)
	account := s.mapAccountNumber(vars["account"])
	group := vars["group"]
	fsid := vars["id"]
	user := vars["user"]

	role := fmt.Sprintf("arn:aws:iam::%s:role/%s", account, s.session.RoleName)

	// IAM doesn't support resource tags, so we can't pass the s.orgPolicy here
	session, err := s.assumeRole(
		r.Context(),
		s.session.ExternalID,
		role,
		"",
		"arn:aws:iam::aws:policy/AmazonElasticFileSystemReadOnlyAccess",
		"arn:aws:iam::aws:policy/IAMReadOnlyAccess",
	)
	if err != nil {
		msg := fmt.Sprintf("failed to assume role in account: %s", account)
		handleError(w, apierror.New(apierror.ErrForbidden, msg, nil))
		return
	}

	efsService := efs.New(efs.WithSession(session.Session))
	iamService := iam.New(iam.WithSession(session.Session))

	orch := newUserOrchestrator(iamService, efsService, s.org)

	out, err := orch.getFilesystemUser(r.Context(), account, group, fsid, user)
	if err != nil {
		handleError(w, err)
		return
	}

	j, err := json.Marshal(out)
	if err != nil {
		log.Errorf("cannot marshal reasponse(%v) into JSON: %s", out, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(j)
}
