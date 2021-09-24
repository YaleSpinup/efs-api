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
	session         *session.Session
	Service         efsiface.EFSAPI
	DefaultKmsKeyId string
	DefaultSgs      []string
	DefaultSubnets  []string
}

type EFSOption func(*EFS)

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

func New(opts ...EFSOption) EFS {
	e := EFS{}

	for _, opt := range opts {
		opt(&e)
	}

	if e.session != nil {
		e.Service = efs.New(e.session)
	}

	return e
}

func WithSession(sess *session.Session) EFSOption {
	return func(e *EFS) {
		log.Debug("using aws session")
		e.session = sess
	}
}

func WithCredentials(key, secret, token, region string) EFSOption {
	return func(e *EFS) {
		log.Debugf("creating new session with key id %s in region %s", key, region)
		sess := session.Must(session.NewSession(&aws.Config{
			Credentials: credentials.NewStaticCredentials(key, secret, token),
			Region:      aws.String(region),
		}))
		e.session = sess
	}
}

func WithDefaultKMSKeyId(keyId string) EFSOption {
	return func(e *EFS) {
		log.Debugf("using default kms keyid %s", keyId)
		e.DefaultKmsKeyId = keyId
	}
}

func WithDefaultSgs(sgs []string) EFSOption {
	return func(e *EFS) {
		log.Debugf("using default security groups %+v", sgs)
		e.DefaultSgs = sgs
	}
}

func WithDefaultSubnets(subnets []string) EFSOption {
	return func(e *EFS) {
		log.Debugf("using default subnets %+v", subnets)
		e.DefaultSubnets = subnets
	}
}
