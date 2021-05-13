package ec2

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/YaleSpinup/efs-api/common"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
)

var testTime = time.Now()

// mockEC2Client is a fake EC2 client
type mockEC2Client struct {
	ec2iface.EC2API
	t   *testing.T
	err error
}

func newmockEC2Client(t *testing.T, err error) ec2iface.EC2API {
	return &mockEC2Client{
		t:   t,
		err: err,
	}
}

func TestNewSession(t *testing.T) {
	e := NewSession(common.Account{})
	to := reflect.TypeOf(e).String()
	if to != "ec2.EC2" {
		t.Errorf("expected type to be 'ec2.EC2', got %s", to)
	}
}

func (m *mockEC2Client) DescribeSubnetsWithContext(ctx context.Context, input *ec2.DescribeSubnetsInput, opts ...request.Option) (*ec2.DescribeSubnetsOutput, error) {
	if m.err != nil {
		return nil, m.err
	}

	for _, s := range input.SubnetIds {
		if aws.StringValue(s) == "multisubnet" {
			return &ec2.DescribeSubnetsOutput{
				Subnets: []*ec2.Subnet{
					{AvailabilityZone: aws.String("az1")},
					{AvailabilityZone: aws.String("az2")},
					{AvailabilityZone: aws.String("az3")},
				},
			}, nil
		}
	}

	return &ec2.DescribeSubnetsOutput{
		Subnets: []*ec2.Subnet{
			{
				AvailabilityZone: aws.String("az1"),
			},
		},
	}, nil
}

func TestEC2_GetSubnet(t *testing.T) {
	type fields struct {
		Service ec2iface.EC2API
	}
	type args struct {
		ctx    context.Context
		subnet string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *ec2.Subnet
		wantErr bool
	}{
		{
			name: "empty input",
			fields: fields{
				Service: newmockEC2Client(t, nil),
			},
			args: args{
				ctx:    context.TODO(),
				subnet: "",
			},
			wantErr: true,
		},
		{
			name: "example input",
			fields: fields{
				Service: newmockEC2Client(t, nil),
			},
			args: args{
				ctx:    context.TODO(),
				subnet: "subnet-01",
			},
			want: &ec2.Subnet{
				AvailabilityZone: aws.String("az1"),
			},
		},
		{
			name: "multi subnet error",
			fields: fields{
				Service: newmockEC2Client(t, nil),
			},
			args: args{
				ctx:    context.TODO(),
				subnet: "multisubnet",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &EC2{
				Service: tt.fields.Service,
			}
			got, err := e.GetSubnet(tt.args.ctx, tt.args.subnet)
			if (err != nil) != tt.wantErr {
				t.Errorf("EC2.GetSubnet() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("EC2.GetSubnet() = %v, want %v", got, tt.want)
			}
		})
	}
}
