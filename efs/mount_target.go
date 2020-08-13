package efs

import (
	"context"

	"github.com/YaleSpinup/apierror"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awsutil"
	"github.com/aws/aws-sdk-go/service/efs"
	log "github.com/sirupsen/logrus"
)

// CreateMountTarget creates an EFS mount target
func (e *EFS) CreateMountTarget(ctx context.Context, input *efs.CreateMountTargetInput) (*efs.MountTargetDescription, error) {
	if input == nil || input.FileSystemId == nil || input.SubnetId == nil {
		return nil, apierror.New(apierror.ErrBadRequest, "invalid input", nil)
	}

	log.Infof("creating efs mount target for fs %s in %s with sgs %+v", aws.StringValue(input.FileSystemId), aws.StringValue(input.SubnetId), aws.StringValueSlice(input.SecurityGroups))

	output, err := e.Service.CreateMountTargetWithContext(ctx, input)
	if err != nil {
		return nil, ErrCode("failed to create mount target", err)
	}

	return output, nil
}

// ListMountTargetsForFileSystem lists the mount targets for a filesystem - there should never be more than 2 since you're only allowed one per AZ
func (e *EFS) ListMountTargetsForFileSystem(ctx context.Context, id string) ([]*efs.MountTargetDescription, error) {
	if id == "" {
		return nil, apierror.New(apierror.ErrBadRequest, "invalid input", nil)
	}

	log.Infof("listing efs mount targets for fs %s", id)

	output, err := e.Service.DescribeMountTargetsWithContext(ctx, &efs.DescribeMountTargetsInput{
		MaxItems:     aws.Int64(100),
		FileSystemId: aws.String(id),
	})
	if err != nil {
		return nil, ErrCode("failed to list mount targets for filesystem", err)
	}

	log.Debugf("got list of mount targets for fs %s: %s", id, awsutil.Prettify(output.MountTargets))

	return output.MountTargets, nil
}

// DeleteMountTarget deletes mount targets
func (e *EFS) DeleteMountTarget(ctx context.Context, id string) error {
	if id == "" {
		return apierror.New(apierror.ErrBadRequest, "invalid input", nil)
	}

	log.Infof("deleting mount target %s", id)

	_, err := e.Service.DeleteMountTargetWithContext(ctx, &efs.DeleteMountTargetInput{
		MountTargetId: aws.String(id),
	})
	if err != nil {
		return ErrCode("failed to delete mount target", err)
	}

	return nil
}
