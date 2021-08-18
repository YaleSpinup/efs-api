package efs

import (
	"context"
	"reflect"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/efs"
	"github.com/aws/aws-sdk-go/service/efs/efsiface"
)

var (
	testAccount = "00112233445566"
	testApArn   = "arn:aws:elasticfilesystem:us-east-1:00112233445566:access-point/fsap-0c487e43029400d26"
	testApId    = "fsap-0c487e43029400d26"
)

func (m *mockEFSClient) CreateAccessPointWithContext(ctx context.Context, input *efs.CreateAccessPointInput, opts ...request.Option) (*efs.CreateAccessPointOutput, error) {
	if m.err != nil {
		return nil, m.err
	}

	return &efs.CreateAccessPointOutput{
		AccessPointArn: aws.String(testApArn),
		AccessPointId:  aws.String(testApId),
		ClientToken:    input.ClientToken,
		FileSystemId:   input.FileSystemId,
		LifeCycleState: aws.String("available"),
		Name:           aws.String("testFilesystem"),
		OwnerId:        aws.String(testAccount),
		PosixUser:      input.PosixUser,
		RootDirectory:  input.RootDirectory,
		Tags:           input.Tags,
	}, nil
}

func (m *mockEFSClient) DescribeAccessPointsWithContext(ctx context.Context, input *efs.DescribeAccessPointsInput, opts ...request.Option) (*efs.DescribeAccessPointsOutput, error) {
	if m.err != nil {
		return nil, m.err
	}

	aps := []*efs.AccessPointDescription{}
	if aws.StringValue(input.FileSystemId) == "fs-123" || aws.StringValue(input.AccessPointId) == testApId {
		aps = append(aps, &efs.AccessPointDescription{
			AccessPointArn: aws.String(testApArn),
			AccessPointId:  aws.String(testApId),
			ClientToken:    aws.String("token"),
			FileSystemId:   aws.String("fs-123"),
			LifeCycleState: aws.String("available"),
			Name:           aws.String("testFilesystem"),
			OwnerId:        aws.String(testAccount),
		})
	}

	return &efs.DescribeAccessPointsOutput{AccessPoints: aps}, nil
}

func (m *mockEFSClient) DeleteAccessPointWithContext(ctx context.Context, input *efs.DeleteAccessPointInput, opts ...request.Option) (*efs.DeleteAccessPointOutput, error) {
	if m.err != nil {
		return nil, m.err
	}

	if aws.StringValue(input.AccessPointId) == testApId {
		return &efs.DeleteAccessPointOutput{}, nil
	}

	return nil, awserr.New(efs.ErrCodeAccessPointNotFound, "not found", nil)
}

func TestEFS_CreateAccessPoint(t *testing.T) {
	type fields struct {
		Service         efsiface.EFSAPI
		DefaultKmsKeyId string
		DefaultSgs      []string
		DefaultSubnets  []string
	}
	type args struct {
		ctx   context.Context
		input *efs.CreateAccessPointInput
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *efs.CreateAccessPointOutput
		wantErr bool
	}{
		{
			name: "nil input",
			fields: fields{
				Service: newMockEFSClient(t, nil),
			},
			args: args{
				ctx:   context.TODO(),
				input: nil,
			},
			wantErr: true,
		},
		{
			name: "min input",
			fields: fields{
				Service: newMockEFSClient(t, nil),
			},
			args: args{
				ctx: context.TODO(),
				input: &efs.CreateAccessPointInput{
					ClientToken:  aws.String("token"),
					FileSystemId: aws.String("fs-123"),
				},
			},
			want: &efs.CreateAccessPointOutput{
				AccessPointArn: aws.String(testApArn),
				AccessPointId:  aws.String(testApId),
				ClientToken:    aws.String("token"),
				FileSystemId:   aws.String("fs-123"),
				LifeCycleState: aws.String("available"),
				Name:           aws.String("testFilesystem"),
				OwnerId:        aws.String(testAccount),
			},
		},
		{
			name: "error from aws",
			fields: fields{
				Service: newMockEFSClient(t, awserr.New(efs.ErrCodeAccessPointAlreadyExists, "exists", nil)),
			},
			args: args{
				ctx: context.TODO(),
				input: &efs.CreateAccessPointInput{
					ClientToken:  aws.String("token"),
					FileSystemId: aws.String("fs-123"),
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &EFS{
				Service:         tt.fields.Service,
				DefaultKmsKeyId: tt.fields.DefaultKmsKeyId,
				DefaultSgs:      tt.fields.DefaultSgs,
				DefaultSubnets:  tt.fields.DefaultSubnets,
			}
			got, err := e.CreateAccessPoint(tt.args.ctx, tt.args.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("EFS.CreateAccessPoint() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("EFS.CreateAccessPoint() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEFS_ListAccessPoints(t *testing.T) {
	type fields struct {
		Service         efsiface.EFSAPI
		DefaultKmsKeyId string
		DefaultSgs      []string
		DefaultSubnets  []string
	}
	type args struct {
		ctx  context.Context
		fsid string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    []*efs.AccessPointDescription
		wantErr bool
	}{
		{
			name: "empty input",
			fields: fields{
				Service: newMockEFSClient(t, nil),
			},
			args: args{
				ctx:  context.TODO(),
				fsid: "",
			},
			wantErr: true,
		},
		{
			name: "min input",
			fields: fields{
				Service: newMockEFSClient(t, nil),
			},
			args: args{
				ctx:  context.TODO(),
				fsid: "fs-123",
			},
			want: []*efs.AccessPointDescription{
				{
					AccessPointArn: aws.String(testApArn),
					AccessPointId:  aws.String(testApId),
					ClientToken:    aws.String("token"),
					FileSystemId:   aws.String("fs-123"),
					LifeCycleState: aws.String("available"),
					Name:           aws.String("testFilesystem"),
					OwnerId:        aws.String(testAccount),
				},
			},
		},
		{
			name: "error from aws",
			fields: fields{
				Service: newMockEFSClient(t, awserr.New(efs.ErrCodeFileSystemNotFound, "not found", nil)),
			},
			args: args{
				ctx:  context.TODO(),
				fsid: "fs-123",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := EFS{
				Service:         tt.fields.Service,
				DefaultKmsKeyId: tt.fields.DefaultKmsKeyId,
				DefaultSgs:      tt.fields.DefaultSgs,
				DefaultSubnets:  tt.fields.DefaultSubnets,
			}
			got, err := e.ListAccessPoints(tt.args.ctx, tt.args.fsid)
			if (err != nil) != tt.wantErr {
				t.Errorf("EFS.ListAccessPoints() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("EFS.ListAccessPoints() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEFS_GetAccessPoint(t *testing.T) {
	type fields struct {
		Service         efsiface.EFSAPI
		DefaultKmsKeyId string
		DefaultSgs      []string
		DefaultSubnets  []string
	}
	type args struct {
		ctx  context.Context
		apid string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *efs.AccessPointDescription
		wantErr bool
	}{
		{
			name: "empty input",
			fields: fields{
				Service: newMockEFSClient(t, nil),
			},
			args: args{
				ctx:  context.TODO(),
				apid: "",
			},
			wantErr: true,
		},
		{
			name: "min input",
			fields: fields{
				Service: newMockEFSClient(t, nil),
			},
			args: args{
				ctx:  context.TODO(),
				apid: testApId,
			},
			want: &efs.AccessPointDescription{
				AccessPointArn: aws.String(testApArn),
				AccessPointId:  aws.String(testApId),
				ClientToken:    aws.String("token"),
				FileSystemId:   aws.String("fs-123"),
				LifeCycleState: aws.String("available"),
				Name:           aws.String("testFilesystem"),
				OwnerId:        aws.String(testAccount),
			},
		},
		{
			name: "error from aws",
			fields: fields{
				Service: newMockEFSClient(t, awserr.New(efs.ErrCodeFileSystemNotFound, "not found", nil)),
			},
			args: args{
				ctx:  context.TODO(),
				apid: testApId,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := EFS{
				Service:         tt.fields.Service,
				DefaultKmsKeyId: tt.fields.DefaultKmsKeyId,
				DefaultSgs:      tt.fields.DefaultSgs,
				DefaultSubnets:  tt.fields.DefaultSubnets,
			}
			got, err := e.GetAccessPoint(tt.args.ctx, tt.args.apid)
			if (err != nil) != tt.wantErr {
				t.Errorf("EFS.GetAccessPoint() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("EFS.GetAccessPoint() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEFS_DeleteAccessPoint(t *testing.T) {
	type fields struct {
		Service         efsiface.EFSAPI
		DefaultKmsKeyId string
		DefaultSgs      []string
		DefaultSubnets  []string
	}
	type args struct {
		ctx  context.Context
		apid string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "empty input",
			fields: fields{
				Service: newMockEFSClient(t, nil),
			},
			args: args{
				ctx:  context.TODO(),
				apid: "",
			},
			wantErr: true,
		},
		{
			name: "min input",
			fields: fields{
				Service: newMockEFSClient(t, nil),
			},
			args: args{
				ctx:  context.TODO(),
				apid: testApId,
			},
		},
		{
			name: "error from aws",
			fields: fields{
				Service: newMockEFSClient(t, awserr.New(efs.ErrCodeFileSystemNotFound, "not found", nil)),
			},
			args: args{
				ctx:  context.TODO(),
				apid: testApId,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := EFS{
				Service:         tt.fields.Service,
				DefaultKmsKeyId: tt.fields.DefaultKmsKeyId,
				DefaultSgs:      tt.fields.DefaultSgs,
				DefaultSubnets:  tt.fields.DefaultSubnets,
			}
			if err := e.DeleteAccessPoint(tt.args.ctx, tt.args.apid); (err != nil) != tt.wantErr {
				t.Errorf("EFS.DeleteAccessPoint() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
