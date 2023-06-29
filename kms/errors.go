package kms

import (
	"github.com/YaleSpinup/apierror"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

func ErrCode(msg string, err error) error {
	if aerr, ok := errors.Cause(err).(awserr.Error); ok {
		switch aerr.Code() {
		case
			// Access forbidden.
			"Forbidden":

			return apierror.New(apierror.ErrForbidden, msg, aerr)
		case
			// Conflict
			"Conflict":

			return apierror.New(apierror.ErrConflict, msg, aerr)
		case
			// Not found.
			"NotFound":

			return apierror.New(apierror.ErrNotFound, msg, aerr)
		case
			"Bad Request":

			return apierror.New(apierror.ErrBadRequest, msg, aerr)
		case
			// Limit Exceeded
			"LimitExceeded":

			return apierror.New(apierror.ErrLimitExceeded, msg, aerr)
		case
			// Service Unavailable
			"ServiceUnavailable":

			return apierror.New(apierror.ErrServiceUnavailable, msg, aerr)
		default:
			m := msg + ": " + aerr.Message()
			return apierror.New(apierror.ErrBadRequest, m, aerr)
		}
	}

	log.Warnf("uncaught error: %s, returning Internal Server Error", err)
	return apierror.New(apierror.ErrInternalError, msg, err)
}
