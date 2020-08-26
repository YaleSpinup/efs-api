package efs

import (
	"context"
	"math/rand"
	"net"
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
)

type subnet struct {
	availabilityZoneId   string
	availabilityZoneName string
	network              string
	subnetId             string
	vpcId                string
}

var testSubnets = map[string]*subnet{
	"subnet-0000001": {
		availabilityZoneId:   "use1-az1",
		availabilityZoneName: "us-east-1a",
		network:              "10.1.0.0/24",
		subnetId:             "subnet-0000001",
		vpcId:                "vpc-0000001",
	},
	"subnet-0000002": {
		availabilityZoneId:   "use1-az2",
		availabilityZoneName: "us-east-1b",
		network:              "10.2.0.0/24",
		subnetId:             "subnet-0000002",
		vpcId:                "vpc-0000001",
	},
	"subnet-0000003": {
		availabilityZoneId:   "use1-az1",
		availabilityZoneName: "us-east-1a",
		network:              "10.3.0.0/24",
		subnetId:             "subnet-0000003",
		vpcId:                "vpc-0000002",
	},
	"subnet-0000004": {
		availabilityZoneId:   "use1-az2",
		availabilityZoneName: "us-east-1b",
		network:              "10.4.0.0/24",
		subnetId:             "subnet-0000004",
		vpcId:                "vpc-0000002",
	},
	"subnet-0000005": {
		availabilityZoneId:   "use1-az1",
		availabilityZoneName: "us-east-1a",
		network:              "10.5.0.0/24",
		subnetId:             "subnet-0000005",
		vpcId:                "vpc-0000003",
	},
}

var testMountTargets = []*efs.MountTargetDescription{
	{
		AvailabilityZoneId:   aws.String("use1-az1"),
		AvailabilityZoneName: aws.String("us-east-1a"),
		FileSystemId:         aws.String("fs-1234567"),
		IpAddress:            aws.String("10.1.0.123"),
		LifeCycleState:       aws.String("created"),
		MountTargetId:        aws.String("fsmt-00112233445566aa"),
		NetworkInterfaceId:   aws.String("eni-00112233445566aa"),
		OwnerId:              aws.String("1234567890"),
		SubnetId:             aws.String("subnet-0000001"),
		VpcId:                aws.String("vpc-0000001"),
	},
	{
		AvailabilityZoneId:   aws.String("use1-az2"),
		AvailabilityZoneName: aws.String("us-east-1b"),
		FileSystemId:         aws.String("fs-1234567"),
		IpAddress:            aws.String("10.2.0.123"),
		LifeCycleState:       aws.String("created"),
		MountTargetId:        aws.String("fsmt-00112233445566dd"),
		NetworkInterfaceId:   aws.String("eni-00112233445566dd"),
		OwnerId:              aws.String("1234567890"),
		SubnetId:             aws.String("subnet-0000002"),
		VpcId:                aws.String("vpc-0000001"),
	},
	{
		AvailabilityZoneId:   aws.String("use1-az1"),
		AvailabilityZoneName: aws.String("us-east-1a"),
		FileSystemId:         aws.String("fs-abcdefg"),
		IpAddress:            aws.String("10.3.0.123"),
		LifeCycleState:       aws.String("created"),
		MountTargetId:        aws.String("fsmt-00112233445566aa"),
		NetworkInterfaceId:   aws.String("eni-00112233445566aa"),
		OwnerId:              aws.String("1234567890"),
		SubnetId:             aws.String("subnet-0000003"),
		VpcId:                aws.String("vpc-0000002"),
	},
	{
		AvailabilityZoneId:   aws.String("use1-az2"),
		AvailabilityZoneName: aws.String("us-east-1b"),
		FileSystemId:         aws.String("fs-abcdefg"),
		IpAddress:            aws.String("10.4.0.123"),
		LifeCycleState:       aws.String("created"),
		MountTargetId:        aws.String("fsmt-00112233445566dd"),
		NetworkInterfaceId:   aws.String("eni-00112233445566dd"),
		OwnerId:              aws.String("1234567890"),
		SubnetId:             aws.String("subnet-0000004"),
		VpcId:                aws.String("vpc-0000002"),
	},
}

func (m *mockEFSClient) DescribeMountTargetsWithContext(ctx context.Context, input *efs.DescribeMountTargetsInput, opts ...request.Option) (*efs.DescribeMountTargetsOutput, error) {
	if m.err != nil {
		return nil, m.err
	}

	if input.AccessPointId != nil {
		return nil, awserr.New(efs.ErrCodeAccessPointNotFound, "not supported", nil)
	}

	if input.MountTargetId != nil {
		for _, mt := range testMountTargets {
			if aws.StringValue(mt.MountTargetId) == aws.StringValue(input.MountTargetId) {
				return &efs.DescribeMountTargetsOutput{
					MountTargets: []*efs.MountTargetDescription{mt},
				}, nil
			}
		}
	}

	if input.FileSystemId != nil {
		mts := []*efs.MountTargetDescription{}
		for _, mt := range testMountTargets {
			if input.FileSystemId != nil {
				if aws.StringValue(mt.FileSystemId) == aws.StringValue(input.FileSystemId) {
					mts = append(mts, mt)
				}
			}
		}
		return &efs.DescribeMountTargetsOutput{MountTargets: mts}, nil
	}

	return &efs.DescribeMountTargetsOutput{
		MountTargets: testMountTargets,
	}, nil
}

func (m *mockEFSClient) CreateMountTargetWithContext(ctx context.Context, input *efs.CreateMountTargetInput, opts ...request.Option) (*efs.MountTargetDescription, error) {
	if m.err != nil {
		return nil, m.err
	}

	snet, ok := testSubnets[aws.StringValue(input.SubnetId)]
	if !ok {
		return nil, awserr.New(efs.ErrCodeSubnetNotFound, "Couldn't create mount target, subnet not found", nil)
	}

	if aws.StringValue(input.IpAddress) == "" {
		ip, nw, err := net.ParseCIDR(snet.network)
		if err != nil {
			return nil, awserr.New(efs.ErrCodeInternalServerError, "failed to parse cidr for subnet", nil)
		}

		for _, mt := range testMountTargets {
			i := net.ParseIP(aws.StringValue(mt.IpAddress))
			if nw.Contains(i) {
				return mt, nil
			}
		}

		input.IpAddress = aws.String(ip.String())
	}

	mtId := strconv.Itoa(rand.Intn(1000000))
	output := &efs.MountTargetDescription{
		AvailabilityZoneId:   aws.String(snet.availabilityZoneId),
		AvailabilityZoneName: aws.String(snet.availabilityZoneName),
		FileSystemId:         input.FileSystemId,
		IpAddress:            input.IpAddress,
		LifeCycleState:       aws.String("creating"),
		MountTargetId:        aws.String("fsmt-" + mtId),
		NetworkInterfaceId:   aws.String("eni-" + mtId),
		OwnerId:              aws.String(uuid.New().String()),
		SubnetId:             aws.String(snet.subnetId),
		VpcId:                aws.String(snet.vpcId),
	}

	return output, nil
}

func (m *mockEFSClient) DeleteMountTargetWithContext(ctx context.Context, input *efs.DeleteMountTargetInput, opts ...request.Option) (*efs.DeleteMountTargetOutput, error) {
	if m.err != nil {
		return nil, m.err
	}

	if aws.StringValue(input.MountTargetId) == "fsmt-012345" {
		return &efs.DeleteMountTargetOutput{}, nil
	}

	return nil, awserr.New(efs.ErrCodeMountTargetNotFound, "mount target not found", nil)
}

func TestCreateMountTarget(t *testing.T) {
	e := EFS{Service: newMockEFSClient(t, nil)}

	if _, err := e.CreateMountTarget(context.TODO(), nil); err == nil {
		t.Error("expected error for nil input, got nil")
	}

	if _, err := e.CreateMountTarget(context.TODO(), &efs.CreateMountTargetInput{
		FileSystemId: nil,
		SubnetId:     aws.String("subnet-foobar"),
	}); err == nil {
		t.Error("expected error for nil FileSystemId, got nil")
	}

	if _, err := e.CreateMountTarget(context.TODO(), &efs.CreateMountTargetInput{
		FileSystemId: aws.String("fs-foobar"),
		SubnetId:     nil,
	}); err == nil {
		t.Error("expected error for nil SubnetId, got nil")
	}

	for _, mt := range testMountTargets {
		input := efs.CreateMountTargetInput{
			FileSystemId: mt.FileSystemId,
			SecurityGroups: []*string{
				aws.String("sg-00000001"),
				aws.String("sg-00000002"),
			},
			SubnetId: mt.SubnetId,
		}

		out, err := e.CreateMountTarget(context.TODO(), &input)
		if err != nil {
			t.Errorf("expected nil error, got %s", err)
		}

		t.Logf("sent input %s, got output %s", awsutil.Prettify(input), awsutil.Prettify(out))

		if !awsutil.DeepEqual(mt, out) {
			t.Errorf("expected %+v, got %+v", awsutil.Prettify(mt), awsutil.Prettify(out))
		}
	}

	// good new mount target
	expect := &efs.MountTargetDescription{
		AvailabilityZoneId:   aws.String("use1-az1"),
		AvailabilityZoneName: aws.String("us-east-1a"),
		FileSystemId:         aws.String("fs-12345"),
		IpAddress:            aws.String("10.5.0.0"),
		LifeCycleState:       aws.String("creating"),
		MountTargetId:        aws.String("fsmt-902081"),
		SubnetId:             aws.String("subnet-0000005"),
		VpcId:                aws.String("vpc-0000003"),
	}
	out, err := e.CreateMountTarget(context.TODO(), &efs.CreateMountTargetInput{
		FileSystemId:   aws.String("fs-12345"),
		SecurityGroups: aws.StringSlice([]string{"sg-12345"}),
		SubnetId:       aws.String("subnet-0000005"),
	})
	if err != nil {
		t.Errorf("expected nil error got %s", err)
	}

	// update expected output with random ids
	expect.OwnerId = out.OwnerId
	expect.MountTargetId = out.MountTargetId
	expect.NetworkInterfaceId = out.NetworkInterfaceId

	if !awsutil.DeepEqual(expect, out) {
		t.Errorf("expected %+v, got %+v", awsutil.Prettify(expect), awsutil.Prettify(out))
	}

	if _, err = e.CreateMountTarget(context.TODO(), &efs.CreateMountTargetInput{
		FileSystemId:   aws.String("fs-12345"),
		SecurityGroups: aws.StringSlice([]string{"sg-12345"}),
		SubnetId:       aws.String("subnet-missing"),
	}); err == nil {
		t.Error("expected error got nil")
	}

}

func TestListMountTargetsForFileSystem(t *testing.T) {
	e := EFS{Service: newMockEFSClient(t, nil)}

	if _, err := e.ListMountTargetsForFileSystem(context.TODO(), ""); err == nil {
		t.Error("expected error for empty input, got nil")
	}

	expectedOut := map[string][]*efs.MountTargetDescription{}
	for _, mt := range testMountTargets {
		fsid := aws.StringValue(mt.FileSystemId)
		if _, ok := expectedOut[fsid]; !ok {
			expectedOut[fsid] = []*efs.MountTargetDescription{}
		}
		expectedOut[fsid] = append(expectedOut[fsid], mt)
	}

	for id, mt := range expectedOut {
		t.Logf("testing list mount targets for fsid %s", id)

		out, err := e.ListMountTargetsForFileSystem(context.TODO(), id)
		if err != nil {
			t.Errorf("expected nil error, got %s", err)
		}

		if !awsutil.DeepEqual(mt, out) {
			t.Errorf("for id %s, expected %+v, got %+v", id, awsutil.Prettify(mt), awsutil.Prettify(out))
		}
	}

	e.Service.(*mockEFSClient).err = awserr.New(efs.ErrCodeBadRequest, "bad request", nil)
	_, err := e.ListMountTargetsForFileSystem(context.TODO(), "fs-123")
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrBadRequest {
			t.Errorf("expected error code %s, got: %s", apierror.ErrBadRequest, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	e.Service.(*mockEFSClient).err = awserr.New(efs.ErrCodeAccessPointNotFound, "not found", nil)
	_, err = e.ListMountTargetsForFileSystem(context.TODO(), "fs-123")
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrNotFound {
			t.Errorf("expected error code %s, got: %s", apierror.ErrNotFound, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	e.Service.(*mockEFSClient).err = awserr.New(efs.ErrCodeMountTargetNotFound, "not found", nil)
	_, err = e.ListMountTargetsForFileSystem(context.TODO(), "fs-123")
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrNotFound {
			t.Errorf("expected error code %s, got: %s", apierror.ErrNotFound, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	e.Service.(*mockEFSClient).err = awserr.New(efs.ErrCodeFileSystemNotFound, "not found", nil)
	_, err = e.ListMountTargetsForFileSystem(context.TODO(), "fs-123")
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrNotFound {
			t.Errorf("expected error code %s, got: %s", apierror.ErrNotFound, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	e.Service.(*mockEFSClient).err = awserr.New(efs.ErrCodeInternalServerError, "internal error", nil)
	_, err = e.ListMountTargetsForFileSystem(context.TODO(), "fs-123")
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrServiceUnavailable {
			t.Errorf("expected error code %s, got: %s", apierror.ErrServiceUnavailable, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}
}

func TestDeleteMountTarget(t *testing.T) {
	e := EFS{Service: newMockEFSClient(t, nil)}

	if err := e.DeleteMountTarget(context.TODO(), "fsmt-012345"); err != nil {
		t.Errorf("expected nil error, not %s", err)
	}

	if err := e.DeleteMountTarget(context.TODO(), ""); err == nil {
		t.Error("expected error for empty input, got nil")
	}

	e.Service.(*mockEFSClient).err = awserr.New(efs.ErrCodeBadRequest, "bad request", nil)
	err := e.DeleteMountTarget(context.TODO(), "fsmt-123")
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrBadRequest {
			t.Errorf("expected error code %s, got: %s", apierror.ErrBadRequest, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	e.Service.(*mockEFSClient).err = awserr.New(efs.ErrCodeInternalServerError, "internal error", nil)
	err = e.DeleteMountTarget(context.TODO(), "fsmt-123")
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrServiceUnavailable {
			t.Errorf("expected error code %s, got: %s", apierror.ErrServiceUnavailable, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	e.Service.(*mockEFSClient).err = awserr.New(efs.ErrCodeDependencyTimeout, "bad request", nil)
	err = e.DeleteMountTarget(context.TODO(), "fsmt-123")
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrServiceUnavailable {
			t.Errorf("expected error code %s, got: %s", apierror.ErrServiceUnavailable, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	e.Service.(*mockEFSClient).err = awserr.New(efs.ErrCodeMountTargetNotFound, "bad request", nil)
	err = e.DeleteMountTarget(context.TODO(), "fsmt-123")
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrNotFound {
			t.Errorf("expected error code %s, got: %s", apierror.ErrNotFound, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}
}
