package efs

import (
	"context"

	"github.com/YaleSpinup/apierror"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/efs"
	log "github.com/sirupsen/logrus"
)

func (e *EFS) CreateAccessPoint(ctx context.Context, input *efs.CreateAccessPointInput) (*efs.CreateAccessPointOutput, error) {
	if input == nil || input.FileSystemId == nil {
		return nil, apierror.New(apierror.ErrBadRequest, "invalid input", nil)
	}

	log.Infof("creating access point for %s", aws.StringValue(input.FileSystemId))

	out, err := e.Service.CreateAccessPointWithContext(ctx, input)
	if err != nil {
		return nil, ErrCode("failed to create access point", err)
	}

	log.Debugf("got output creating access point for %s: %+v", aws.StringValue(input.FileSystemId), out)

	return out, nil
}

func (e EFS) ListAccessPoints(ctx context.Context, fsid string) ([]*efs.AccessPointDescription, error) {
	if fsid == "" {
		return nil, apierror.New(apierror.ErrBadRequest, "invalid input", nil)
	}

	input := efs.DescribeAccessPointsInput{
		FileSystemId: aws.String(fsid),
	}

	log.Infof("getting list of access points %s", fsid)

	output := []*efs.AccessPointDescription{}
	for {
		out, err := e.Service.DescribeAccessPointsWithContext(ctx, &input)
		if err != nil {
			return nil, ErrCode("failed to get access points", err)
		}

		output = append(output, out.AccessPoints...)

		if out.NextToken == nil {
			break
		}

		input.NextToken = out.NextToken
	}

	log.Debugf("got output listing access points: %+v", output)

	return output, nil
}

func (e EFS) GetAccessPoint(ctx context.Context, apid string) (*efs.AccessPointDescription, error) {
	if apid == "" {
		return nil, apierror.New(apierror.ErrBadRequest, "invalid input", nil)
	}

	out, err := e.Service.DescribeAccessPointsWithContext(ctx, &efs.DescribeAccessPointsInput{
		AccessPointId: aws.String(apid),
	})
	if err != nil {
		return nil, ErrCode("failed to get access point", err)
	}

	log.Debugf("got output getting access point: %+v", out)

	if len(out.AccessPoints) > 1 || len(out.AccessPoints) == 0 {
		return nil, apierror.New(apierror.ErrBadRequest, "unexpected number of access points", nil)
	}

	return out.AccessPoints[0], nil
}

func (e EFS) DeleteAccessPoint(ctx context.Context, apid string) error {
	if apid == "" {
		return apierror.New(apierror.ErrBadRequest, "invalid input", nil)
	}

	if _, err := e.Service.DeleteAccessPointWithContext(ctx, &efs.DeleteAccessPointInput{
		AccessPointId: aws.String(apid),
	}); err != nil {
		return ErrCode("failed to delete access point", err)
	}

	return nil
}
