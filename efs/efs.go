package efs

import (
	"github.com/YaleSpinup/efs-api/common"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/efs"
	"github.com/aws/aws-sdk-go/service/efs/efsiface"
	log "github.com/sirupsen/logrus"
)

// EFS is a wrapper around the aws EFS service with some default config info
type EFS struct {
	Service         efsiface.EFSAPI
	DefaultKmsKeyId string
	DefaultSgs      []string
	DefaultSubnets  []string
}

// NewSession creates a new EFS session
func NewSession(account common.Account) EFS {
	log.Infof("creating new aws session for EFS with key id %s in region %s", account.Akid, account.Region)

	s := EFS{}
	config := aws.Config{
		Credentials: credentials.NewStaticCredentials(account.Akid, account.Secret, ""),
		Region:      aws.String(account.Region),
	}

	sess := session.Must(session.NewSession(&config))
	s.Service = efs.New(sess)
	s.DefaultKmsKeyId = account.DefaultKmsKeyId
	s.DefaultSgs = account.DefaultSgs
	s.DefaultSubnets = account.DefaultSubnets

	return s
}
