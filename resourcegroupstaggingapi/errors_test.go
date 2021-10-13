package resourcegroupstaggingapi

import (
	"testing"

	"github.com/YaleSpinup/apierror"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/resourcegroupstaggingapi"
	"github.com/pkg/errors"
)

func TestErrCode(t *testing.T) {
	apiErrorTestCases := map[string]string{
		"": apierror.ErrBadRequest,
		resourcegroupstaggingapi.ErrCodeInternalServiceException:        apierror.ErrInternalError,
		resourcegroupstaggingapi.ErrCodeConcurrentModificationException: apierror.ErrConflict,
		resourcegroupstaggingapi.ErrCodeThrottledException:              apierror.ErrConflict,
		resourcegroupstaggingapi.ErrCodeConstraintViolationException:    apierror.ErrBadRequest,
		resourcegroupstaggingapi.ErrCodeInvalidParameterException:       apierror.ErrBadRequest,
		resourcegroupstaggingapi.ErrCodePaginationTokenExpiredException: apierror.ErrBadRequest,
	}

	for awsErr, apiErr := range apiErrorTestCases {
		expected := apierror.New(apiErr, "test error", awserr.New(awsErr, awsErr, nil))
		err := ErrCode("test error", awserr.New(awsErr, awsErr, nil))

		aerr, ok := err.(apierror.Error)
		if !ok {
			t.Errorf("expected resourcegroupstaggingapi error %s to be an apierror.Error %s, got %s", awsErr, apiErr, err)
		}

		if aerr.String() != expected.String() {
			t.Errorf("expected error '%s', got '%s'", expected, aerr)
		}
	}

	err := ErrCode("test error", errors.New("Unknown"))
	if aerr, ok := errors.Cause(err).(apierror.Error); ok {
		t.Logf("got apierror '%s'", aerr)
	} else {
		t.Errorf("expected unknown error to be an apierror.ErrInternalError, got %s", err)
	}
}
