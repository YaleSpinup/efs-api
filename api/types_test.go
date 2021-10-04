package api

import (
	"reflect"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awsutil"
	"github.com/aws/aws-sdk-go/service/efs"
)

func TestFileSystemResponseFromEFS(t *testing.T) {
	t.Log("TODO")
}

func TestNormalizeTags(t *testing.T) {
	s := server{
		org: "testOrg",
	}

	type testTags struct {
		name   string
		group  string
		tags   []*Tag
		expect []*Tag
	}

	tests := []*testTags{
		{
			name:  "",
			group: "",
			tags:  nil,
			expect: []*Tag{
				{Key: "Name", Value: ""},
				{Key: "spinup:org", Value: "testOrg"},
				{Key: "spinup:spaceid", Value: ""},
			},
		},
		{
			name:  "SomeFS",
			group: "MySpace",
			tags:  nil,
			expect: []*Tag{
				{Key: "Name", Value: "SomeFS"},
				{Key: "spinup:org", Value: "testOrg"},
				{Key: "spinup:spaceid", Value: "MySpace"},
			},
		},
		{
			name:  "SomeFS1",
			group: "MySpace",
			tags: []*Tag{
				{Key: "Name", Value: "SomeOtherFSName"},
				{Key: "spinup:org", Value: "SomeOtherOrgName"},
				{Key: "spinup:spaceid", Value: "SomeOtherSpaceID"},
			},
			expect: []*Tag{
				{Key: "Name", Value: "SomeFS1"},
				{Key: "spinup:org", Value: "testOrg"},
				{Key: "spinup:spaceid", Value: "MySpace"},
			},
		},
	}

	for _, test := range tests {
		out := normalizeTags(s.org, test.name, test.group, test.tags)
		if !reflect.DeepEqual(test.expect, out) {
			t.Errorf("expected %+v, got %+v", awsutil.Prettify(test.expect), awsutil.Prettify(out))
		}
	}
}

func TestFromEFSTags(t *testing.T) {
	input := []*efs.Tag{}
	expected := []*Tag{}
	if output := fromEFSTags(input); !reflect.DeepEqual(expected, output) {
		t.Errorf("expected %+v, got %+v for input %+v", expected, output, input)
	}

	input = []*efs.Tag{
		{
			Key:   aws.String("foo"),
			Value: aws.String("bar"),
		},
	}
	expected = []*Tag{
		{
			Key:   "foo",
			Value: "bar",
		},
	}
	if output := fromEFSTags(input); !reflect.DeepEqual(expected, output) {
		t.Errorf("expected %+v, got %+v for input %+v", expected, output, input)
	}
}

func TestToEFSTags(t *testing.T) {
	input := []*Tag{}
	expected := []*efs.Tag{}
	if output := toEFSTags(input); !reflect.DeepEqual(expected, output) {
		t.Errorf("expected %+v, got %+v for input %+v", expected, output, input)
	}

	input = []*Tag{
		{
			Key:   "foo",
			Value: "bar",
		},
	}
	expected = []*efs.Tag{
		{
			Key:   aws.String("foo"),
			Value: aws.String("bar"),
		},
	}
	if output := toEFSTags(input); !reflect.DeepEqual(expected, output) {
		t.Errorf("expected %+v, got %+v for input %+v", expected, output, input)
	}
}

func TestListFileSystems(t *testing.T) {
	t.Log("TODO")
}

func TestFileSystemExists(t *testing.T) {
	t.Log("TODO")
}
