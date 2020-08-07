package efs

import (
	"reflect"
	"testing"
	"time"

	"github.com/YaleSpinup/efs-api/common"
	"github.com/aws/aws-sdk-go/service/efs/efsiface"
)

var testTime = time.Now()

// mockS3Client is a fake S3 client
type mockEFSClient struct {
	efsiface.EFSAPI
	t   *testing.T
	err error
}

func newMockEFSClient(t *testing.T, err error) efsiface.EFSAPI {
	return &mockEFSClient{
		t:   t,
		err: err,
	}
}

func TestNewSession(t *testing.T) {
	e := NewSession(common.Account{})
	to := reflect.TypeOf(e).String()
	if to != "efs.EFS" {
		t.Errorf("expected type to be 'efs.EFS', got %s", to)
	}
}
