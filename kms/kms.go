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
	"strings"
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

// isRequiredKmsTagKey given a search value check an array of keys for a match
func (k *KMS) isRequiredKmsTagKey(search string, keys []string) bool {
	for _, key := range keys {
		if key == search {
			return true
		}
	}

	return false
}

// isPartnerTagKey is the given tag key a partner key
func (k *KMS) isPartnerTagKey(tagKey string) bool {
	return strings.Split(tagKey, ":")[2] == "partner"
}

// isOrgTagKey is the given tag key an org key
func (k *KMS) isOrgTagKey(tagKey string) bool {
	return strings.Split(tagKey, ":")[2] == "org"
}

// GetKmsKeyIdByTags Get the required kms key id for the account by its tags
func (k *KMS) GetKmsKeyIdByTags(ctx context.Context, inputTags []string, org string) (string, error) {
	if len(inputTags) == 0 {
		return "", apierror.New(apierror.ErrBadRequest, "empty kms key input tags", nil)
	}
	log.Info("searching for required kms key via tags...")

	accountKeyIds, err := k.ListKmsKeyIds(ctx)
	if err != nil {
		return "", err
	}

	var targetKeyId string
	for _, keyId := range accountKeyIds {
		matchingTagsCount := 0
		tags, err := k.GetKmsKeyTags(ctx, keyId)
		if err != nil {
			return "", apierror.New(apierror.ErrBadRequest, "there was an error fetching the kms key tags", nil)
		}

		for _, tag := range tags {
			tagKey := aws.StringValue(tag.TagKey)
			tagValue := aws.StringValue(tag.TagValue)

			if k.isRequiredKmsTagKey(tagKey, inputTags) {
				if (k.isOrgTagKey(tagKey) && tagValue == org) || k.isPartnerTagKey(tagKey) {
					matchingTagsCount++
				}
			}
		}

		if matchingTagsCount == len(inputTags) {
			targetKeyId = keyId
		}
	}

	log.Info("found target kms key id for the account")

	return targetKeyId, nil
}

// GetKmsKeyTags get the kms key tags for a specific kms key id
func (k *KMS) GetKmsKeyTags(ctx context.Context, keyId string) ([]*kms.Tag, error) {
	log.Infof("fetching tags for kms key id: %s", keyId)
	tags, _ := k.Service.ListResourceTagsWithContext(ctx, &kms.ListResourceTagsInput{
		KeyId: aws.String(keyId),
	})

	return tags.Tags, nil
}

// ListKmsKeyIds fetch the kms key ids for an account
func (k *KMS) ListKmsKeyIds(ctx context.Context) ([]string, error) {
	log.Infof("fetching kms key ids for the account")
	keysListOutput, err := k.Service.ListKeysWithContext(ctx, &kms.ListKeysInput{})
	if err != nil {
		return []string{}, err
	}

	var accountKeyIds []string
	for _, key := range keysListOutput.Keys {
		accountKeyIds = append(accountKeyIds, aws.StringValue(key.KeyId))
	}

	return accountKeyIds, nil
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
