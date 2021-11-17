package api

import (
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/efs"
	"github.com/aws/aws-sdk-go/service/iam"
	log "github.com/sirupsen/logrus"
)

// Tag is an API tag
type Tag struct {
	Key   string
	Value string
}

// normalizeTags strips the org, spaceid and name from the given tags and ensures they
// are set to the API org and the group string, name passed to the request.  it also
// skips any aws specific tags
func normalizeTags(org, name, group string, tags []*Tag) []*Tag {
	normalizedTags := []*Tag{}
	for _, t := range tags {
		if t.Key == "spinup:spaceid" || t.Key == "spinup:org" || t.Key == "Name" {
			continue
		}

		if strings.HasPrefix(t.Key, "aws:") {
			continue
		}

		normalizedTags = append(normalizedTags, t)
	}

	normalizedTags = append(normalizedTags,
		&Tag{
			Key:   "Name",
			Value: name,
		},
		&Tag{
			Key:   "spinup:org",
			Value: org,
		},
		&Tag{
			Key:   "spinup:spaceid",
			Value: group,
		})

	log.Debugf("returning normalized tags: %+v", normalizedTags)
	return normalizedTags
}

// fromEFSTags converts from EFS tags to api Tags
func fromEFSTags(efsTags []*efs.Tag) []*Tag {
	tags := make([]*Tag, 0, len(efsTags))
	for _, t := range efsTags {
		tags = append(tags, &Tag{
			Key:   aws.StringValue(t.Key),
			Value: aws.StringValue(t.Value),
		})
	}
	return tags
}

// toEFSTags converts from api Tags to EFS tags
func toEFSTags(tags []*Tag) []*efs.Tag {
	efsTags := make([]*efs.Tag, 0, len(tags))
	for _, t := range tags {
		efsTags = append(efsTags, &efs.Tag{
			Key:   aws.String(t.Key),
			Value: aws.String(t.Value),
		})
	}
	return efsTags
}

// fromIAMTags converts from IAM tags to api Tags
func fromIAMTags(iamTags []*iam.Tag) []*Tag {
	tags := make([]*Tag, 0, len(iamTags))
	for _, t := range iamTags {
		tags = append(tags, &Tag{
			Key:   aws.StringValue(t.Key),
			Value: aws.StringValue(t.Value),
		})
	}
	return tags
}

// toIAMTags converts from api Tags to IAM tags
func toIAMTags(tags []*Tag) []*iam.Tag {
	iamTags := make([]*iam.Tag, 0, len(tags))
	for _, t := range tags {
		iamTags = append(iamTags, &iam.Tag{
			Key:   aws.String(t.Key),
			Value: aws.String(t.Value),
		})
	}
	return iamTags
}
