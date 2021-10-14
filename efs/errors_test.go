package efs

import (
	"testing"

	"github.com/YaleSpinup/apierror"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/efs"
	"github.com/pkg/errors"
)

func TestErrCode(t *testing.T) {
	apiErrorTestCases := map[string]string{
		"":        apierror.ErrBadRequest,
		"unknonw": apierror.ErrBadRequest,

		"Forbidden": apierror.ErrForbidden,

		efs.ErrCodeAccessPointAlreadyExists: apierror.ErrConflict,
		efs.ErrCodeFileSystemAlreadyExists:  apierror.ErrConflict,
		efs.ErrCodeFileSystemInUse:          apierror.ErrConflict,
		efs.ErrCodeIpAddressInUse:           apierror.ErrConflict,
		efs.ErrCodeMountTargetConflict:      apierror.ErrConflict,
		"Conflict":                          apierror.ErrConflict,

		efs.ErrCodeAccessPointNotFound: apierror.ErrNotFound,
		efs.ErrCodeFileSystemNotFound:  apierror.ErrNotFound,
		efs.ErrCodeMountTargetNotFound: apierror.ErrNotFound,
		"NotFound":                     apierror.ErrNotFound,

		efs.ErrCodeBadRequest:                        apierror.ErrBadRequest,
		efs.ErrCodeIncorrectFileSystemLifeCycleState: apierror.ErrBadRequest,
		efs.ErrCodeIncorrectMountTargetState:         apierror.ErrBadRequest,
		efs.ErrCodeInsufficientThroughputCapacity:    apierror.ErrBadRequest,
		efs.ErrCodeInvalidPolicyException:            apierror.ErrBadRequest,
		efs.ErrCodePolicyNotFound:                    apierror.ErrBadRequest,
		efs.ErrCodeSecurityGroupLimitExceeded:        apierror.ErrBadRequest,
		efs.ErrCodeSecurityGroupNotFound:             apierror.ErrBadRequest,
		efs.ErrCodeSubnetNotFound:                    apierror.ErrBadRequest,
		efs.ErrCodeUnsupportedAvailabilityZone:       apierror.ErrBadRequest,
		efs.ErrCodeValidationException:               apierror.ErrBadRequest,

		efs.ErrCodeAccessPointLimitExceeded:      apierror.ErrLimitExceeded,
		efs.ErrCodeFileSystemLimitExceeded:       apierror.ErrLimitExceeded,
		efs.ErrCodeNetworkInterfaceLimitExceeded: apierror.ErrLimitExceeded,
		efs.ErrCodeNoFreeAddressesInSubnet:       apierror.ErrLimitExceeded,
		efs.ErrCodeThroughputLimitExceeded:       apierror.ErrLimitExceeded,
		efs.ErrCodeTooManyRequests:               apierror.ErrLimitExceeded,
		"LimitExceeded":                          apierror.ErrLimitExceeded,

		efs.ErrCodeDependencyTimeout:   apierror.ErrServiceUnavailable,
		efs.ErrCodeInternalServerError: apierror.ErrServiceUnavailable,
		"ServiceUnavailable":           apierror.ErrServiceUnavailable,
	}

	for awsErr, apiErr := range apiErrorTestCases {
		expected := apierror.New(apiErr, "test error", awserr.New(awsErr, awsErr, nil))
		err := ErrCode("test error", awserr.New(awsErr, awsErr, nil))

		var aerr apierror.Error
		if !errors.As(err, &aerr) {
			t.Errorf("expected aws error %s to be an apierror.Error %s, got %s", awsErr, apiErr, err)
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
