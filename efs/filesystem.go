package efs

import (
	"context"
	"fmt"

	"github.com/YaleSpinup/apierror"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awsutil"
	"github.com/aws/aws-sdk-go/service/efs"
	log "github.com/sirupsen/logrus"
)

// CreateFilesystem creates an EFS filesystem
func (e *EFS) CreateFileSystem(ctx context.Context, input *efs.CreateFileSystemInput) (*efs.FileSystemDescription, error) {
	if input == nil {
		return nil, apierror.New(apierror.ErrBadRequest, "invalid input", nil)
	}

	log.Infof("creating efs filesystem with input %+v", awsutil.Prettify(input))

	output, err := e.Service.CreateFileSystemWithContext(ctx, input)
	if err != nil {
		return nil, ErrCode("failed to create filesystem", err)
	}

	return output, nil
}

// DeleteFilsystem deletes an EFS filesystem
func (e *EFS) DeleteFileSystem(ctx context.Context, id string) error {
	if id == "" {
		return apierror.New(apierror.ErrBadRequest, "invalid input", nil)
	}

	log.Infof("deleting efs filesystem %s", id)

	if _, err := e.Service.DeleteFileSystemWithContext(ctx, &efs.DeleteFileSystemInput{
		FileSystemId: aws.String(id),
	}); err != nil {
		return ErrCode("failed to delete filesystem", err)
	}

	return nil
}

// ListFilesystems lists all filesystems if nil input is passed
func (e *EFS) ListFileSystems(ctx context.Context, input *efs.DescribeFileSystemsInput) ([]string, error) {
	if input == nil {
		return nil, apierror.New(apierror.ErrBadRequest, "invalid input", nil)
	}

	log.Infof("listing efs filesystems with input %s", awsutil.Prettify(input))

	input.MaxItems = aws.Int64(100)
	output := []string{}
	for {
		out, err := e.Service.DescribeFileSystemsWithContext(ctx, input)
		if err != nil {
			return nil, ErrCode("failed to list filesystems", err)
		}

		for _, f := range out.FileSystems {
			output = append(output, aws.StringValue(f.FileSystemId))
		}

		if out.NextMarker == nil {
			break
		}
		input.Marker = out.NextMarker
	}

	log.Debugf("got list of filsystems: %+v", output)
	return output, nil
}

// GetFilesystem gets details about a filesystem
func (e *EFS) GetFileSystem(ctx context.Context, id string) (*efs.FileSystemDescription, error) {
	if id == "" {
		return nil, apierror.New(apierror.ErrBadRequest, "invalid input", nil)
	}

	log.Infof("getting details for efs filesystem %s", id)

	output, err := e.Service.DescribeFileSystemsWithContext(ctx, &efs.DescribeFileSystemsInput{
		FileSystemId: aws.String(id),
	})
	if err != nil {
		return nil, ErrCode("failed to get filesystem", err)
	}

	if len(output.FileSystems) == 0 {
		msg := fmt.Sprintf("%s not found", id)
		return nil, apierror.New(apierror.ErrNotFound, msg, nil)
	}

	if num := len(output.FileSystems); num > 1 {
		msg := fmt.Sprintf("unexpected number of filesystems found for id %s (%d)", id, num)
		return nil, apierror.New(apierror.ErrInternalError, msg, nil)
	}

	return output.FileSystems[0], nil
}
