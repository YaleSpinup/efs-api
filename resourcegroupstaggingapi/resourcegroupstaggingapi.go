package resourcegroupstaggingapi

import (
	"context"
	"strings"

	"github.com/YaleSpinup/apierror"
	"github.com/YaleSpinup/efs-api/common"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/resourcegroupstaggingapi"
	"github.com/aws/aws-sdk-go/service/resourcegroupstaggingapi/resourcegroupstaggingapiiface"
	log "github.com/sirupsen/logrus"
)

// ResourceGroupsTaggingAPI is a wrapper around the aws resourcegroupstaggingapi service with some default config info
type ResourceGroupsTaggingAPI struct {
	Service resourcegroupstaggingapiiface.ResourceGroupsTaggingAPIAPI
}

// Tag Filter is used to filter resources based on tags.  The Value portion is optional.
type TagFilter struct {
	Key   string
	Value []string
}

// NewSession creates a new cloudfront session
func NewSession(account common.Account) ResourceGroupsTaggingAPI {
	s := ResourceGroupsTaggingAPI{}
	log.Infof("creating new aws session for resourcegroupstaggingapi with key id %s in region %s", account.Akid, account.Region)
	sess := session.Must(session.NewSession(&aws.Config{
		Credentials: credentials.NewStaticCredentials(account.Akid, account.Secret, ""),
		Region:      aws.String(account.Region),
	}))
	s.Service = resourcegroupstaggingapi.New(sess)
	return s
}

// GetResourcesWithTags returns all of the resources with a type in the list of types that matches the tagfilters.  More
// details about which services support the resourgroup tagging api is here https://docs.aws.amazon.com/ARG/latest/userguide/supported-resources.html
func (r *ResourceGroupsTaggingAPI) GetResourcesWithTags(ctx context.Context, types []string, filters []*TagFilter) ([]*resourcegroupstaggingapi.ResourceTagMapping, error) {
	if len(filters) == 0 {
		return nil, apierror.New(apierror.ErrBadRequest, "invalid input", nil)
	}

	log.Infof("getting resources with type '%s' that match tags", strings.Join(types, ", "))

	tagFilters := make([]*resourcegroupstaggingapi.TagFilter, 0, len(filters))
	for _, f := range filters {
		log.Debugf("tagfilter: %s:%+v", f.Key, f.Value)
		tagFilters = append(tagFilters, &resourcegroupstaggingapi.TagFilter{
			Key:    aws.String(f.Key),
			Values: aws.StringSlice(f.Value),
		})
	}

	out, err := r.Service.GetResourcesWithContext(ctx, &resourcegroupstaggingapi.GetResourcesInput{
		ResourceTypeFilters: aws.StringSlice(types),
		TagFilters:          tagFilters,
	})
	if err != nil {
		return nil, ErrCode("getting resource with tags", err)
	}

	log.Debugf("got output from get resources: %+v", out)

	return out.ResourceTagMappingList, nil
}
