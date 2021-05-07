package ec2

import (
	"context"
	"fmt"

	"github.com/YaleSpinup/apierror"
	"github.com/YaleSpinup/efs-api/common"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
	log "github.com/sirupsen/logrus"
)

// EC2 is a wrapper around the aws EC2 service with some default config info
type EC2 struct {
	Service ec2iface.EC2API
}

// NewSession creates a new EFS session
func NewSession(account common.Account) EC2 {
	log.Infof("creating new aws session for EC2 with key id %s in region %s", account.Akid, account.Region)

	s := EC2{}
	config := aws.Config{
		Credentials: credentials.NewStaticCredentials(account.Akid, account.Secret, ""),
		Region:      aws.String(account.Region),
	}

	sess := session.Must(session.NewSession(&config))
	s.Service = ec2.New(sess)

	return s
}

func (e *EC2) GetSubnet(ctx context.Context, subnet string) (*ec2.Subnet, error) {
	if subnet == "" {
		return nil, apierror.New(apierror.ErrBadRequest, "invalid input", nil)
	}

	log.Infof("getting details about subnet %s", subnet)

	out, err := e.Service.DescribeSubnetsWithContext(ctx, &ec2.DescribeSubnetsInput{
		SubnetIds: aws.StringSlice([]string{subnet}),
	})
	if err != nil {
		return nil, ErrCode("failed to describe subnet", err)
	}

	log.Debugf("got output describing subnet %s: %+v", subnet, out)

	if length := len(out.Subnets); length != 1 {
		return nil, apierror.New(apierror.ErrBadRequest, fmt.Sprintf("unexpected subnet list length %d", length), nil)
	}

	return out.Subnets[0], nil
}
