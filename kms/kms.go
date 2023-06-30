package kms

import (
	"context"
	"fmt"
	"strings"

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

// hasInputTags given a search value check an array of keys for a match
func (k *KMS) hasInputTags(search string, keys []string) bool {
	for _, key := range keys {
		if key == search {
			return true
		}
	}

	return false
}

func (k *KMS) GetKmsKeyIdByTags(ctx context.Context, inputTags []string, org string) (string, error) {
	if len(inputTags) == 0 {
		return "", apierror.New(apierror.ErrBadRequest, "empty kms key input tags", nil)
	}

	log.Info("retrieving list of keys to check for tags")

	output, err := k.Service.ListKeysWithContext(ctx, &kms.ListKeysInput{})
	if err != nil {
		return "", err
	}

	log.Infof("retrieved %d keys", len(output.Keys))
	log.Infof("searching for kms keys matching the tags %v", inputTags)

	var targetKeyId string
	for _, key := range output.Keys {
		keyId := aws.StringValue(key.KeyId)
		tags, err := k.Service.ListResourceTagsWithContext(ctx, &kms.ListResourceTagsInput{KeyId: key.KeyId})
		if err != nil {
			return "", err
		}

		var foundKeys []string
		for _, tag := range tags.Tags {
			tagKey := aws.StringValue(tag.TagKey)
			tagValue := aws.StringValue(tag.TagValue)

			if k.hasInputTags(tagKey, inputTags) {
				keyValueEnd := strings.Split(tagKey, ":")[2]

				if keyValueEnd != "org" || keyValueEnd == "org" && tagValue == org {
					foundKeys = append(foundKeys, tagKey)
				}

				if len(foundKeys) == len(inputTags) {
					targetKeyId = keyId
				}
			}
		}
	}

	if targetKeyId == "" {
		return "", apierror.New(
			apierror.ErrNotFound,
			fmt.Sprintf("cannot find kms key with tag key %s", inputTags),
			nil,
		)
	}

	return targetKeyId, nil
}

func (k *KMS) GetKmsKeyId(ctx context.Context, aliasName string) (string, error) {
	if aliasName == "" {
		return "", apierror.New(apierror.ErrBadRequest, "empty alias input", nil)
	}

	input := &kms.ListAliasesInput{}
	output, err := k.Service.ListAliasesWithContext(ctx, input)
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
