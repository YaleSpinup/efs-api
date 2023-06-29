package api

import (
	"context"
	"fmt"
	"time"

	"github.com/YaleSpinup/apierror"
	yefs "github.com/YaleSpinup/efs-api/efs"
	ykms "github.com/YaleSpinup/efs-api/kms"
	"github.com/YaleSpinup/flywheel"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/efs"
	"github.com/google/uuid"
)

func (s server) accessPointCreate(ctx context.Context, account, group, fsid string, req *AccessPointCreateRequest) (*AccessPoint, *flywheel.Task, error) {
	acctNum := s.mapAccountNumber(account)
	role := fmt.Sprintf("arn:aws:iam::%s:role/%s", acctNum, s.session.RoleName)
	policy, err := generatePolicy("elasticfilesystem:*", "kms:*")
	if err != nil {
		return nil, nil, apierror.New(apierror.ErrNotFound, "cannot generate policy", nil)
	}

	session, err := s.assumeRole(
		ctx,
		s.session.ExternalID,
		role,
		policy,
	)
	if err != nil {
		return nil, nil, apierror.New(apierror.ErrNotFound, "failed to assume role in account", nil)
	}

	kmsService := ykms.New(ykms.WithSession(session.Session))
	defaultKmsKey, err := kmsService.GetKmsKeyId(ctx, s.getKMSKeyAlias(account))
	if err != nil {
		return nil, nil, apierror.New(apierror.ErrInternalError, "failed to get KMS Key", nil)
	}

	service := yefs.New(yefs.WithSession(session.Session),
		yefs.WithDefaultKMSKeyId(acctNum, defaultKmsKey),
		yefs.WithDefaultSgs(s.efsServices[account].DefaultSgs),
		yefs.WithDefaultSubnets(s.efsServices[account].DefaultSubnets))

	filesystem, err := service.GetFileSystem(ctx, fsid)
	if err != nil {
		return nil, nil, err
	}

	// if the Name is empty, just make one up
	if req.Name == "" {
		req.Name = uuid.NewString()
	}

	// using append here in case the filesystem as no tags
	tags := []*efs.Tag{}
	for _, t := range filesystem.Tags {
		if aws.StringValue(t.Key) == "Name" {
			name := fmt.Sprintf("%s-%s", aws.StringValue(filesystem.Name), req.Name)
			tags = append(tags, &efs.Tag{
				Key:   aws.String("Name"),
				Value: aws.String(name),
			})
			continue
		}
		tags = append(tags, t)
	}

	// generate a new task to track and start it
	task := flywheel.NewTask()
	input := efs.CreateAccessPointInput{
		ClientToken:   aws.String(task.ID),
		FileSystemId:  aws.String(fsid),
		PosixUser:     req.PosixUser,
		RootDirectory: req.RootDirectory,
		Tags:          tags,
	}

	out, err := service.CreateAccessPoint(ctx, &input)
	if err != nil {
		return nil, nil, err
	}

	// start the async orchestration to wait for access point to become available
	go func() {
		fsid := aws.StringValue(filesystem.FileSystemId)
		apid := aws.StringValue(out.AccessPointId)

		fsCtx, cancel := context.WithCancel(context.Background())
		defer cancel()

		msgChan, errChan := s.startTask(fsCtx, task)

		msgChan <- fmt.Sprintf("requested creation of accesspoint for filesystem %s", fsid)

		// wait for the accesspoint to become available
		if err = retry(10, 2*time.Second, func() error {
			msg := fmt.Sprintf("checking if accesspoint %s is available before continuing", apid)
			msgChan <- msg

			out, err := service.GetAccessPoint(fsCtx, apid)
			if err != nil {
				msgChan <- fmt.Sprintf("got error checking if accesspoint %s is available: %s", apid, err)
				return err
			}

			if status := aws.StringValue(out.LifeCycleState); status != "available" {
				msgChan <- fmt.Sprintf("accesspoint %s is not yet available (%s)", apid, status)
				return fmt.Errorf("accesspoint %s not yet available", apid)
			}

			msgChan <- fmt.Sprintf("accesspoint %s is available", apid)
			return nil
		}); err != nil {
			errChan <- fmt.Errorf("failed to create access point %s for filesystem %s, timeout waiting to become available: %s", apid, fsid, err.Error())
			return
		}

	}()

	ap := &AccessPoint{
		AccessPointArn: aws.StringValue(out.AccessPointArn),
		AccessPointId:  aws.StringValue(out.AccessPointId),
		LifeCycleState: aws.StringValue(out.LifeCycleState),
		Name:           aws.StringValue(out.Name),
		PosixUser:      out.PosixUser,
		RootDirectory:  out.RootDirectory,
	}

	return ap, task, nil
}

func (s *server) listFilesystemAccessPoints(ctx context.Context, account, group, fsid string) ([]string, error) {
	acctNum := s.mapAccountNumber(account)
	role := fmt.Sprintf("arn:aws:iam::%s:role/%s", acctNum, s.session.RoleName)
	policy, err := generatePolicy("elasticfilesystem:*", "kms:*")
	if err != nil {
		return nil, err
	}

	session, err := s.assumeRole(
		ctx,
		s.session.ExternalID,
		role,
		policy,
	)
	if err != nil {
		return nil, err
	}

	kmsService := ykms.New(ykms.WithSession(session.Session))
	defaultKmsKey, err := kmsService.GetKmsKeyId(ctx, s.getKMSKeyAlias(account))
	if err != nil {
		return nil, apierror.New(apierror.ErrInternalError, "failed to get KMS Key", nil)
	}

	service := yefs.New(yefs.WithSession(session.Session),
		yefs.WithDefaultKMSKeyId(acctNum, defaultKmsKey),
		yefs.WithDefaultSgs(s.efsServices[account].DefaultSgs),
		yefs.WithDefaultSubnets(s.efsServices[account].DefaultSubnets))

	out, err := service.ListAccessPoints(ctx, fsid)
	if err != nil {
		return nil, err
	}

	output := []string{}
	for _, ap := range out {
		output = append(output, aws.StringValue(ap.AccessPointId))
	}

	return output, nil
}

func (s *server) getFilesystemAccessPoint(ctx context.Context, account, group, fsid, apid string) (*AccessPoint, error) {
	acctNum := s.mapAccountNumber(account)
	role := fmt.Sprintf("arn:aws:iam::%s:role/%s", acctNum, s.session.RoleName)
	policy, err := generatePolicy("elasticfilesystem:*", "kms:*")
	if err != nil {
		return nil, err
	}

	session, err := s.assumeRole(
		ctx,
		s.session.ExternalID,
		role,
		policy,
	)
	if err != nil {
		return nil, err
	}

	kmsService := ykms.New(ykms.WithSession(session.Session))
	defaultKmsKey, err := kmsService.GetKmsKeyId(ctx, s.getKMSKeyAlias(account))
	if err != nil {
		return nil, apierror.New(apierror.ErrInternalError, "failed to get KMS Key", nil)
	}

	service := yefs.New(yefs.WithSession(session.Session),
		yefs.WithDefaultKMSKeyId(acctNum, defaultKmsKey),
		yefs.WithDefaultSgs(s.efsServices[account].DefaultSgs),
		yefs.WithDefaultSubnets(s.efsServices[account].DefaultSubnets))

	out, err := service.GetAccessPoint(ctx, apid)
	if err != nil {
		return nil, err
	}

	return accessPointResponseFromEFS(out), nil
}

func (s *server) deleteFilesystemAccessPoint(ctx context.Context, account, group, fsid, apid string) error {
	acctNum := s.mapAccountNumber(account)
	role := fmt.Sprintf("arn:aws:iam::%s:role/%s", acctNum, s.session.RoleName)
	policy, err := generatePolicy("elasticfilesystem:*", "kms:*")
	if err != nil {
		return err
	}

	session, err := s.assumeRole(
		ctx,
		s.session.ExternalID,
		role,
		policy,
	)
	if err != nil {
		return err
	}

	kmsService := ykms.New(ykms.WithSession(session.Session))
	defaultKmsKey, err := kmsService.GetKmsKeyId(ctx, s.getKMSKeyAlias(account))
	if err != nil {
		return apierror.New(apierror.ErrInternalError, "failed to get KMS Key", nil)
	}

	service := yefs.New(yefs.WithSession(session.Session),
		yefs.WithDefaultKMSKeyId(acctNum, defaultKmsKey),
		yefs.WithDefaultSgs(s.efsServices[account].DefaultSgs),
		yefs.WithDefaultSubnets(s.efsServices[account].DefaultSubnets))

	if err := service.DeleteAccessPoint(ctx, apid); err != nil {
		return err
	}

	return nil
}
