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

func (o *userOrchestrator) createFilesystemUser(ctx context.Context, account, group, fsid string, req *FileSystemUserCreateRequest) (*FileSystemUserResponse, error) {
	filesystem, err := o.efsClient.GetFileSystem(ctx, fsid)
	if err != nil {
		return nil, err
	}

	name := aws.StringValue(filesystem.Name)
	path := fmt.Sprintf("/spinup/%s/%s/%s/", o.org, group, name)
	userName := fmt.Sprintf("%s-%s", name, req.UserName)

	// set the user tags from the filesystems
	tags := normalizeTags(o.org, userName, group, fromEFSTags(filesystem.Tags))
	tags = append(tags, &Tag{
		Key:   "ResourceName",
		Value: aws.StringValue(filesystem.Name),
	})

	user, err := o.iamClient.CreateUser(ctx, userName, path, toIAMTags(tags))
	if err != nil {
		return nil, err
	}

	if err := o.iamClient.WaitForUser(ctx, userName); err != nil {
		return nil, err
	}

	grp := fmt.Sprintf("%s-%s", "SpinupEFSAdminGroup", o.org)

	if err := o.iamClient.AddUserToGroup(ctx, userName, grp); err != nil {
		return nil, err
	}

	return filesystemUserResponseFromIAM(o.org, user, nil), nil
}

// deleteFilesystemUser deletes a filesystem user and all associated access keys
func (o *userOrchestrator) deleteFilesystemUser(ctx context.Context, account, group, fsid, user string) error {
	filesystem, err := o.efsClient.GetFileSystem(ctx, fsid)
	if err != nil {
		return err
	}
	name := aws.StringValue(filesystem.Name)

	path := fmt.Sprintf("/spinup/%s/%s/%s/", o.org, group, name)
	userName := fmt.Sprintf("%s-%s", name, user)

	if _, err := o.iamClient.GetUserWithPath(ctx, path, userName); err != nil {
		return err
	}

	groups, err := o.iamClient.ListGroupsForUser(ctx, userName)
	if err != nil {
		return err
	}

	for _, g := range groups {
		if err := o.iamClient.RemoveUserFromGroup(ctx, userName, g); err != nil {
			return err
		}
	}

	keys, err := o.iamClient.ListAccessKeys(ctx, userName)
	if err != nil {
		return err
	}

	for _, k := range keys {
		if err := o.iamClient.DeleteAccessKey(ctx, userName, aws.StringValue(k.AccessKeyId)); err != nil {
			return err
		}
	}

	if err := o.iamClient.DeleteUser(ctx, userName); err != nil {
		return err
	}

	return nil
}

// deleteAllFilesystemUsers deletes all users for a repository
func (o *userOrchestrator) deleteAllFilesystemUsers(ctx context.Context, account, group, fsid string) ([]string, error) {
	// list all users for the repository
	users, err := o.listFilesystemUsers(ctx, account, group, fsid)
	if err != nil {
		return nil, err
	}

	for _, u := range users {
		if err := o.deleteFilesystemUser(ctx, account, group, fsid, u); err != nil {
			log.Errorf("failed to delete filesystem %s user %s: %s", fsid, u, err)
		}
	}

	return users, nil
}

// listFilesystemUsers lists the IAM users in a path specific to the filesystem
func (o *userOrchestrator) listFilesystemUsers(ctx context.Context, account, group, fsid string) ([]string, error) {
	filesystem, err := o.efsClient.GetFileSystem(ctx, fsid)
	if err != nil {
		return nil, err
	}
	name := aws.StringValue(filesystem.Name)

	path := fmt.Sprintf("/spinup/%s/%s/%s/", o.org, group, name)

	users, err := o.iamClient.ListUsers(ctx, path)
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
func (o *userOrchestrator) getFilesystemUser(ctx context.Context, account, group, fsid, user string) (*FileSystemUserResponse, error) {
	filesystem, err := o.efsClient.GetFileSystem(ctx, fsid)
	if err != nil {
		return nil, err
	}
	name := aws.StringValue(filesystem.Name)

	path := fmt.Sprintf("/spinup/%s/%s/%s/", o.org, group, name)
	userName := fmt.Sprintf("%s-%s", name, user)

	iamUser, err := o.iamClient.GetUserWithPath(ctx, path, userName)
	if err != nil {
		return nil, err
	}

	keys, err := o.iamClient.ListAccessKeys(ctx, userName)
	if err != nil {
		return nil, err
	}

	return filesystemUserResponseFromIAM(o.org, iamUser, keys), nil
}

// updateFilesystemUser updates a user for a filesystem
func (o *userOrchestrator) updateFilesystemUser(ctx context.Context, account, group, fsid, user string, req *FileSystemUserUpdateRequest) (*FileSystemUserResponse, error) {
	filesystem, err := o.efsClient.GetFileSystem(ctx, fsid)
	if err != nil {
		return nil, err
	}

	name := aws.StringValue(filesystem.Name)
	userName := fmt.Sprintf("%s-%s", name, user)

	if _, err := o.getFilesystemUser(ctx, account, group, fsid, user); err != nil {
		return nil, err
	}

	response := &FileSystemUserResponse{
		UserName: user,
	}

	if req.ResetKey {
		// get a list of users access keys
		keys, err := o.iamClient.ListAccessKeys(ctx, userName)
		if err != nil {
			return nil, err
		}

		newKeyOut, err := o.iamClient.CreateAccessKey(ctx, userName)
		if err != nil {
			return nil, err
		}
		response.AccessKey = newKeyOut

		deletedKeyIds := make([]string, 0, len(keys))
		// delete the old access keys
		for _, k := range keys {
			if err = o.iamClient.DeleteAccessKey(ctx, userName, aws.StringValue(k.AccessKeyId)); err != nil {
				return response, err
			}
			deletedKeyIds = append(deletedKeyIds, aws.StringValue(k.AccessKeyId))
		}

		response.DeletedAccessKeys = deletedKeyIds
	}

	return response, nil
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
	userName := fmt.Sprintf("%s-%s", name, user)

	tags = normalizeTags(s.org, userName, group, tags)

	if err := iamService.TagUser(ctx, userName, toIAMTags(tags)); err != nil {
		return err
	}

	return nil
}
