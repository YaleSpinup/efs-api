package kms

import (
	"context"

	"github.com/YaleSpinup/apierror"
	"github.com/YaleSpinup/efs-api/common"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/kms"
	log "github.com/sirupsen/logrus"
)

// KMS is a wrapper around the aws KMS service with some default config info
type KMS struct {
	session *session.Session
	Service *kms.KMS
}

type KMSOption func(*KMS)

// NewSession creates a new KMS session
func NewSession(account common.Account) KMS {
	log.Infof("creating new aws session for KMS with key id %s in region %s", account.Akid, account.Region)

	s := KMS{}
	config := aws.Config{
		Credentials: credentials.NewStaticCredentials(account.Akid, account.Secret, ""),
		Region:      aws.String(account.Region),
	}

	sess := session.Must(session.NewSession(&config))
	s.Service = kms.New(sess)

	return s
}

func (k *KMS) GetKmsKeyId(ctx context.Context, aliasName string) (string, error) {
	if aliasName == "" {
		return "", apierror.New(apierror.ErrBadRequest, "invalid input", nil)
	}

	input := &kms.ListAliasesInput{}
	output, err := k.Service.ListAliases(input)
	if err != nil {
		return "", err
	}

	// Find the target key ID by iterating through the aliases
	var targetKeyId string
	for _, alias := range output.Aliases {
		if aws.StringValue(alias.AliasName) == aliasName {
			targetKeyId = aws.StringValue(alias.TargetKeyId)
			break
		}
	}
	return targetKeyId, nil

}

func New(opts ...KMSOption) KMS {
	e := KMS{}

	for _, opt := range opts {
		opt(&e)
	}

	if e.session != nil {
		e.Service = kms.New(e.session)
	}

	return e
}

func WithSession(sess *session.Session) KMSOption {
	return func(e *KMS) {
		log.Debug("using aws session")
		e.session = sess
	}
}

func WithCredentials(key, secret, token, region string) KMSOption {
	return func(e *KMS) {
		log.Debugf("creating new session with key id %s in region %s", key, region)
		sess := session.Must(session.NewSession(&aws.Config{
			Credentials: credentials.NewStaticCredentials(key, secret, token),
			Region:      aws.String(region),
		}))
		e.session = sess
	}
}
