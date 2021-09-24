package api

import (
	"context"
	"fmt"
	"strings"

	"github.com/YaleSpinup/apierror"
	"github.com/YaleSpinup/aws-go/services/iam"
	"github.com/YaleSpinup/efs-api/efs"
	"github.com/aws/aws-sdk-go/aws"
	log "github.com/sirupsen/logrus"
)

func (s *server) createFilesystemUser(ctx context.Context, account, group, fsid string, req *FileSystemUserCreateRequest) (*FileSystemUserResponse, error) {
	if req.UserName == "" {
		return nil, apierror.New(apierror.ErrBadRequest, "Username is a required field", nil)
	}

	log.Infof("creating filesystem %s user %s", fsid, req.UserName)

	role := fmt.Sprintf("arn:aws:iam::%s:role/%s", account, s.session.RoleName)

	policy, err := s.filesystemUserCreatePolicy()
	if err != nil {
		return nil, apierror.New(apierror.ErrInternalError, "failed to generate policy", err)
	}

	// IAM doesn't support resource tags, so we can't pass the s.orgPolicy here
	session, err := s.assumeRole(
		ctx,
		s.session.ExternalID,
		role,
		policy,
		"arn:aws:iam::aws:policy/AmazonElasticFileSystemReadOnlyAccess",
	)
	if err != nil {
		msg := fmt.Sprintf("failed to assume role in account: %s", account)
		return nil, apierror.New(apierror.ErrForbidden, msg, nil)
	}

	efsService := efs.New(efs.WithSession(session.Session))
	iamService := iam.New(iam.WithSession(session.Session))

	filesystem, err := efsService.GetFileSystem(ctx, fsid)
	if err != nil {
		return nil, err
	}

	name := aws.StringValue(filesystem.Name)
	path := fmt.Sprintf("/spinup/%s/%s/%s/", s.org, group, name)
	userName := fmt.Sprintf("%s-%s", name, req.UserName)

	tags := normalizeTags(s.org, userName, group, fromEFSTags(filesystem.Tags))

	user, err := iamService.CreateUser(ctx, userName, path, toIAMTags(tags))
	if err != nil {
		return nil, err
	}

	if err := iamService.WaitForUser(ctx, userName); err != nil {
		return nil, err
	}

	return filesystemUserResponseFromIAM(s.org, user, nil), nil
}

// deleteFilesystemUser deletes a filesystem user and all associated access keys
func (s *server) deleteFilesystemUser(ctx context.Context, account, group, fsid, user string) error {
	role := fmt.Sprintf("arn:aws:iam::%s:role/%s", account, s.session.RoleName)

	policy, err := s.filesystemUserDeletePolicy()
	if err != nil {
		return apierror.New(apierror.ErrInternalError, "failed to generate policy", err)
	}

	// IAM doesn't support resource tags, so we can't pass the s.orgPolicy here
	session, err := s.assumeRole(
		ctx,
		s.session.ExternalID,
		role,
		policy,
		"arn:aws:iam::aws:policy/AmazonElasticFileSystemReadOnlyAccess",
	)
	if err != nil {
		msg := fmt.Sprintf("failed to assume role in account: %s", account)
		return apierror.New(apierror.ErrForbidden, msg, nil)
	}

	efsService := efs.New(efs.WithSession(session.Session))
	iamService := iam.New(iam.WithSession(session.Session))

	filesystem, err := efsService.GetFileSystem(ctx, fsid)
	if err != nil {
		return err
	}
	name := aws.StringValue(filesystem.Name)

	path := fmt.Sprintf("/spinup/%s/%s/%s/", s.org, group, name)
	userName := fmt.Sprintf("%s-%s", name, user)

	if _, err := iamService.GetUserWithPath(ctx, path, userName); err != nil {
		return err
	}

	keys, err := iamService.ListAccessKeys(ctx, userName)
	if err != nil {
		return err
	}

	for _, k := range keys {
		if err := iamService.DeleteAccessKey(ctx, userName, aws.StringValue(k.AccessKeyId)); err != nil {
			return err
		}
	}

	if err := iamService.DeleteUser(ctx, userName); err != nil {
		return err
	}

	return nil
}

// deleteAllFilesystemUsers deletes all users for a repository
func (s *server) deleteAllFilesystemUsers(ctx context.Context, account, group, fsid string) ([]string, error) {
	// list all users for the repository
	users, err := s.listFilesystemUsers(ctx, account, group, fsid)
	if err != nil {
		return nil, err
	}

	for _, u := range users {
		if err := s.deleteFilesystemUser(ctx, account, group, fsid, u); err != nil {
			log.Errorf("failed to delete filesystem %s user %s: %s", fsid, u, err)
		}
	}

	return users, nil
}

// listFilesystemUsers lists the IAM users in a path specific to the filesystem
func (s *server) listFilesystemUsers(ctx context.Context, account, group, fsid string) ([]string, error) {
	role := fmt.Sprintf("arn:aws:iam::%s:role/%s", account, s.session.RoleName)

	// IAM doesn't support resource tags, so we can't pass the s.orgPolicy here
	session, err := s.assumeRole(
		ctx,
		s.session.ExternalID,
		role,
		"",
		"arn:aws:iam::aws:policy/IAMReadOnlyAccess",
		"arn:aws:iam::aws:policy/AmazonElasticFileSystemReadOnlyAccess",
	)
	if err != nil {
		msg := fmt.Sprintf("failed to assume role in account: %s", account)
		return nil, apierror.New(apierror.ErrForbidden, msg, nil)
	}

	efsService := efs.New(efs.WithSession(session.Session))
	iamService := iam.New(iam.WithSession(session.Session))

	filesystem, err := efsService.GetFileSystem(ctx, fsid)
	if err != nil {
		return nil, err
	}
	name := aws.StringValue(filesystem.Name)

	path := fmt.Sprintf("/spinup/%s/%s/%s/", s.org, group, name)

	users, err := iamService.ListUsers(ctx, path)
	if err != nil {
		return nil, err
	}

	prefix := name + "-"

	trimmed := make([]string, 0, len(users))
	for _, u := range users {
		log.Debugf("trimming prefix '%s' from username %s", prefix, u)
		u = strings.TrimPrefix(u, prefix)
		trimmed = append(trimmed, u)
	}
	users = trimmed

	return users, nil
}

// getFilesystemUser gets the details about a filesystem user with the username generated from the fs name and the passed username
func (s *server) getFilesystemUser(ctx context.Context, account, group, fsid, user string) (*FileSystemUserResponse, error) {
	role := fmt.Sprintf("arn:aws:iam::%s:role/%s", account, s.session.RoleName)

	// IAM doesn't support resource tags, so we can't pass the s.orgPolicy here
	session, err := s.assumeRole(
		ctx,
		s.session.ExternalID,
		role,
		"",
		"arn:aws:iam::aws:policy/AmazonElasticFileSystemReadOnlyAccess",
		"arn:aws:iam::aws:policy/IAMReadOnlyAccess",
	)
	if err != nil {
		msg := fmt.Sprintf("failed to assume role in account: %s", account)
		return nil, apierror.New(apierror.ErrForbidden, msg, nil)
	}

	efsService := efs.New(efs.WithSession(session.Session))
	iamService := iam.New(iam.WithSession(session.Session))

	filesystem, err := efsService.GetFileSystem(ctx, fsid)
	if err != nil {
		return nil, err
	}
	name := aws.StringValue(filesystem.Name)

	path := fmt.Sprintf("/spinup/%s/%s/%s/", s.org, group, name)
	userName := fmt.Sprintf("%s-%s", name, user)

	iamUser, err := iamService.GetUserWithPath(ctx, path, userName)
	if err != nil {
		return nil, err
	}

	keys, err := iamService.ListAccessKeys(ctx, userName)
	if err != nil {
		return nil, err
	}

	return filesystemUserResponseFromIAM(s.org, iamUser, keys), nil
}

func (s *server) updateTagsForUser(ctx context.Context, account, group, fsid, user string, tags []*Tag) error {
	log.Infof("updating tags for filesystem %s user %s", fsid, user)

	role := fmt.Sprintf("arn:aws:iam::%s:role/%s", account, s.session.RoleName)

	policy, err := s.filesystemUserUpdatePolicy()
	if err != nil {
		return apierror.New(apierror.ErrInternalError, "failed to generate policy", err)
	}

	// IAM doesn't support resource tags, so we can't pass the s.orgPolicy here
	session, err := s.assumeRole(
		ctx,
		s.session.ExternalID,
		role,
		policy,
		"arn:aws:iam::aws:policy/AmazonElasticFileSystemReadOnlyAccess",
	)
	if err != nil {
		msg := fmt.Sprintf("failed to assume role in account: %s", account)
		return apierror.New(apierror.ErrForbidden, msg, nil)
	}

	efsService := efs.New(efs.WithSession(session.Session))
	iamService := iam.New(iam.WithSession(session.Session))

	filesystem, err := efsService.GetFileSystem(ctx, fsid)
	if err != nil {
		return err
	}

	name := aws.StringValue(filesystem.Name)
	// path := fmt.Sprintf("/spinup/%s/%s/%s/", s.org, group, name)
	userName := fmt.Sprintf("%s-%s", name, user)

	tags = normalizeTags(s.org, userName, group, tags)

	if err := iamService.TagUser(ctx, userName, toIAMTags(tags)); err != nil {
		return err
	}

	return nil
}
