package efs

import (
	"context"
	"math/rand"
	"reflect"
	"strconv"
	"testing"

	"github.com/YaleSpinup/apierror"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/awsutil"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/efs"
	"github.com/google/uuid"
	"github.com/pkg/errors"
)

var testFileSystems = []*efs.FileSystemDescription{
	{
		CreationTime:         &testTime,
		CreationToken:        aws.String("xxxxx"),
		Encrypted:            aws.Bool(true),
		FileSystemArn:        aws.String("arn:aws:elasticfilesystem:us-east-1:1111333322228888:file-system/fs-01234567"),
		FileSystemId:         aws.String("fs-01234567"),
		KmsKeyId:             aws.String("arn:aws:kms:us-east-1:1111333322228888:key/0000000-1111-2222-3333-444444444444"),
		LifeCycleState:       aws.String("available"),
		Name:                 aws.String("superfs"),
		NumberOfMountTargets: aws.Int64(0),
		OwnerId:              aws.String("010101010101"),
		PerformanceMode:      aws.String("generalPurpose"),
		SizeInBytes:          &efs.FileSystemSize{},
		Tags: []*efs.Tag{
			{
				Key:   aws.String("Name"),
				Value: aws.String("superfs"),
			},
			{
				Key:   aws.String("foo"),
				Value: aws.String("bar"),
			},
		},
	},
	{
		CreationTime:         &testTime,
		CreationToken:        aws.String("yyyyy"),
		Encrypted:            aws.Bool(true),
		FileSystemArn:        aws.String("arn:aws:elasticfilesystem:us-east-1:1111333322228888:file-system/fs-76543210"),
		FileSystemId:         aws.String("fs-76543210"),
		KmsKeyId:             aws.String("arn:aws:kms:us-east-1:1111333322228888:key/0000000-1111-2222-3333-444444444444"),
		LifeCycleState:       aws.String("available"),
		Name:                 aws.String("boringfs"),
		NumberOfMountTargets: aws.Int64(0),
		OwnerId:              aws.String("010101010101"),
		PerformanceMode:      aws.String("generalPurpose"),
		SizeInBytes:          &efs.FileSystemSize{},
		Tags: []*efs.Tag{
			{
				Key:   aws.String("Name"),
				Value: aws.String("boringfs"),
			},
			{
				Key:   aws.String("foo"),
				Value: aws.String("biz"),
			},
		},
	},
}

func (m *mockEFSClient) CreateFileSystemWithContext(ctx context.Context, input *efs.CreateFileSystemInput, opts ...request.Option) (*efs.FileSystemDescription, error) {
	if m.err != nil {
		return nil, m.err
	}

	var name string
	for _, t := range input.Tags {
		if aws.StringValue(t.Key) == "Name" {
			name = aws.StringValue(t.Value)
		}
	}

	fsid := strconv.Itoa(rand.Intn(1000000))
	owner := uuid.New().String()
	output := &efs.FileSystemDescription{
		CreationTime:                 &testTime,
		CreationToken:                input.CreationToken,
		Encrypted:                    input.Encrypted,
		FileSystemArn:                aws.String("arn:aws:elasticfilesystem:us-east-1:1111333322228888:file-system/fs-" + fsid),
		FileSystemId:                 aws.String("fs-" + fsid),
		KmsKeyId:                     input.KmsKeyId,
		LifeCycleState:               aws.String("creating"),
		Name:                         aws.String(name),
		NumberOfMountTargets:         aws.Int64(0),
		OwnerId:                      aws.String(owner),
		PerformanceMode:              input.PerformanceMode,
		ProvisionedThroughputInMibps: input.ProvisionedThroughputInMibps,
		SizeInBytes:                  &efs.FileSystemSize{},
		ThroughputMode:               input.ThroughputMode,
		Tags:                         input.Tags,
	}

	// if the filesystem exists in the list, override the values from it
	for _, fs := range testFileSystems {
		if aws.StringValue(fs.CreationToken) == aws.StringValue(input.CreationToken) {
			output.FileSystemArn = fs.FileSystemArn
			output.FileSystemId = fs.FileSystemId
			output.LifeCycleState = fs.LifeCycleState
			output.NumberOfMountTargets = fs.NumberOfMountTargets
			output.OwnerId = fs.OwnerId
			output.SizeInBytes = fs.SizeInBytes

			break
		}
	}

	return output, nil
}

func (m *mockEFSClient) DeleteFileSystemWithContext(ctx context.Context, input *efs.DeleteFileSystemInput, opts ...request.Option) (*efs.DeleteFileSystemOutput, error) {
	if m.err != nil {
		return nil, m.err
	}

	for _, fs := range testFileSystems {
		if aws.StringValue(fs.FileSystemId) == aws.StringValue(input.FileSystemId) {
			return &efs.DeleteFileSystemOutput{}, nil
		}
	}

	return nil, awserr.New(efs.ErrCodeFileSystemNotFound, "Couldn't delete filesystem, not found", nil)
}

func (m *mockEFSClient) DescribeFileSystemsWithContext(ctx context.Context, input *efs.DescribeFileSystemsInput, opts ...request.Option) (*efs.DescribeFileSystemsOutput, error) {
	if m.err != nil {
		return nil, m.err
	}

	if input.FileSystemId != nil {
		for _, fs := range testFileSystems {
			if aws.StringValue(fs.FileSystemId) == aws.StringValue(input.FileSystemId) {
				return &efs.DescribeFileSystemsOutput{
					FileSystems: []*efs.FileSystemDescription{fs},
				}, nil
			}
		}

		return nil, awserr.New(efs.ErrCodeFileSystemNotFound, "Couldn't find filesystem, not found", nil)
	}

	return &efs.DescribeFileSystemsOutput{FileSystems: testFileSystems}, nil
}

func TestCreateFileSystem(t *testing.T) {
	e := EFS{Service: newMockEFSClient(t, nil)}

	if _, err := e.CreateFileSystem(context.TODO(), nil); err == nil {
		t.Errorf("expected error for nil input, got %s", err)
	}

	for _, fs := range testFileSystems {
		input := efs.CreateFileSystemInput{
			CreationToken:                fs.CreationToken,
			Encrypted:                    fs.Encrypted,
			KmsKeyId:                     fs.KmsKeyId,
			PerformanceMode:              fs.PerformanceMode,
			ProvisionedThroughputInMibps: fs.ProvisionedThroughputInMibps,
			Tags:                         fs.Tags,
			ThroughputMode:               fs.ThroughputMode,
		}

		out, err := e.CreateFileSystem(context.TODO(), &input)
		if err != nil {
			t.Errorf("expected nil error, got %s", err)
		}

		t.Logf("sent input %s, got output %s", awsutil.Prettify(input), awsutil.Prettify(out))

		if !awsutil.DeepEqual(fs, out) {
			t.Errorf("expected %+v, got %+v", awsutil.Prettify(fs), awsutil.Prettify(out))
		}
	}

	e.Service.(*mockEFSClient).err = awserr.New(efs.ErrCodeBadRequest, "bad request", nil)
	_, err := e.CreateFileSystem(context.TODO(), &efs.CreateFileSystemInput{})
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrBadRequest {
			t.Errorf("expected error code %s, got: %s", apierror.ErrConflict, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	e.Service.(*mockEFSClient).err = awserr.New(efs.ErrCodeInternalServerError, "internal server error", nil)
	_, err = e.CreateFileSystem(context.TODO(), &efs.CreateFileSystemInput{})
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrServiceUnavailable {
			t.Errorf("expected error code %s, got: %s", apierror.ErrServiceUnavailable, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	e.Service.(*mockEFSClient).err = awserr.New(efs.ErrCodeFileSystemAlreadyExists, "already exists", nil)
	_, err = e.CreateFileSystem(context.TODO(), &efs.CreateFileSystemInput{})
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrConflict {
			t.Errorf("expected error code %s, got: %s", apierror.ErrConflict, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	e.Service.(*mockEFSClient).err = awserr.New(efs.ErrCodeFileSystemLimitExceeded, "limit exceeded", nil)
	_, err = e.CreateFileSystem(context.TODO(), &efs.CreateFileSystemInput{})
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrLimitExceeded {
			t.Errorf("expected error code %s, got: %s", apierror.ErrConflict, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	e.Service.(*mockEFSClient).err = awserr.New(efs.ErrCodeInsufficientThroughputCapacity, "insufficient throughput capacity", nil)
	_, err = e.CreateFileSystem(context.TODO(), &efs.CreateFileSystemInput{})
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrBadRequest {
			t.Errorf("expected error code %s, got: %s", apierror.ErrBadRequest, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	e.Service.(*mockEFSClient).err = awserr.New(efs.ErrCodeThroughputLimitExceeded, "throughput limit exceeded", nil)
	_, err = e.CreateFileSystem(context.TODO(), &efs.CreateFileSystemInput{})
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrLimitExceeded {
			t.Errorf("expected error code %s, got: %s", apierror.ErrConflict, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test non-aws error
	e.Service.(*mockEFSClient).err = errors.New("things blowing up!")
	_, err = e.CreateFileSystem(context.TODO(), &efs.CreateFileSystemInput{})
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrInternalError {
			t.Errorf("expected error code %s, got: %s", apierror.ErrInternalError, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}
}

func TestListFileSystems(t *testing.T) {
	e := EFS{Service: newMockEFSClient(t, nil)}

	if _, err := e.ListFileSystems(context.TODO(), nil); err == nil {
		t.Errorf("expected error for nil input, got %s", err)
	}

	expected := []string{}
	for _, fs := range testFileSystems {
		expected = append(expected, aws.StringValue(fs.FileSystemId))
	}

	out, err := e.ListFileSystems(context.TODO(), &efs.DescribeFileSystemsInput{})
	if err != nil {
		t.Errorf("expected nil error, got %s", err)
	}

	if !reflect.DeepEqual(expected, out) {
		t.Errorf("expected %+v, got %+v", expected, out)
	}
}

func TestGetFileSystem(t *testing.T) {
	t.Log("TODO")
}

func TestDeleteFileSystem(t *testing.T) {
	t.Log("TODO")
}
