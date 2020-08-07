package efs

import (
	"github.com/YaleSpinup/apierror"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/efs"
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
			//efs.ErrCodeAccessPointAlreadyExists for service response error code
			// "AccessPointAlreadyExists".
			//
			// Returned if the access point you are trying to create already exists, with
			// the creation token you provided in the request.
			efs.ErrCodeAccessPointAlreadyExists,

			//efs.ErrCodeFileSystemAlreadyExists for service response error code
			// "FileSystemAlreadyExists".
			//
			// Returned if the file system you are trying to create already exists, with
			// the creation token you provided.
			efs.ErrCodeFileSystemAlreadyExists,

			//efs.ErrCodeFileSystemInUse for service response error code
			// "FileSystemInUse".
			//
			// Returned if a file system has mount targets.
			efs.ErrCodeFileSystemInUse,

			//efs.ErrCodeIpAddressInUse for service response error code
			// "IpAddressInUse".
			//
			// Returned if the request specified an IpAddress that is already in use in
			// the subnet.
			efs.ErrCodeIpAddressInUse,

			//efs.ErrCodeMountTargetConflict for service response error code
			// "MountTargetConflict".
			//
			// Returned if the mount target would violate one of the specified restrictions
			// based on the file system's existing mount targets.
			efs.ErrCodeMountTargetConflict,

			// Conflict
			"Conflict":

			return apierror.New(apierror.ErrConflict, msg, aerr)
		case
			//efs.ErrCodeAccessPointNotFound for service response error code
			// "AccessPointNotFound".
			//
			// Returned if the specified AccessPointId value doesn't exist in the requester's
			// AWS account.
			efs.ErrCodeAccessPointNotFound,

			//efs.ErrCodeFileSystemNotFound for service response error code
			// "FileSystemNotFound".
			//
			// Returned if the specified FileSystemId value doesn't exist in the requester's
			// AWS account.
			efs.ErrCodeFileSystemNotFound,

			//efs.ErrCodeMountTargetNotFound for service response error code
			// "MountTargetNotFound".
			//
			// Returned if there is no mount target with the specified ID found in the caller's
			// account.
			efs.ErrCodeMountTargetNotFound,

			// Not found.
			"NotFound":

			return apierror.New(apierror.ErrNotFound, msg, aerr)

		case
			//efs.ErrCodeBadRequest for service response error code
			// "BadRequest".
			//
			// Returned if the request is malformed or contains an error such as an invalid
			// parameter value or a missing required parameter.
			efs.ErrCodeBadRequest,

			//efs.ErrCodeIncorrectFileSystemLifeCycleState for service response error code
			// "IncorrectFileSystemLifeCycleState".
			//
			// Returned if the file system's lifecycle state is not "available".
			efs.ErrCodeIncorrectFileSystemLifeCycleState,

			//efs.ErrCodeIncorrectMountTargetState for service response error code
			// "IncorrectMountTargetState".
			//
			// Returned if the mount target is not in the correct state for the operation.
			efs.ErrCodeIncorrectMountTargetState,

			//efs.ErrCodeInsufficientThroughputCapacity for service response error code
			// "InsufficientThroughputCapacity".
			//
			// Returned if there's not enough capacity to provision additional throughput.
			// This value might be returned when you try to create a file system in provisioned
			// throughput mode, when you attempt to increase the provisioned throughput
			// of an existing file system, or when you attempt to change an existing file
			// system from bursting to provisioned throughput mode.
			efs.ErrCodeInsufficientThroughputCapacity,

			//efs.ErrCodeInvalidPolicyException for service response error code
			// "InvalidPolicyException".
			//
			// Returned if the FileSystemPolicy is is malformed or contains an error such
			// as an invalid parameter value or a missing required parameter. Returned in
			// the case of a policy lockout safety check error.
			efs.ErrCodeInvalidPolicyException,

			//efs.ErrCodePolicyNotFound for service response error code
			// "PolicyNotFound".
			//
			// Returned if the default file system policy is in effect for the EFS file
			// system specified.
			efs.ErrCodePolicyNotFound,

			//efs.ErrCodeSecurityGroupLimitExceeded for service response error code
			// "SecurityGroupLimitExceeded".
			//
			// Returned if the size of SecurityGroups specified in the request is greater
			// than five.
			efs.ErrCodeSecurityGroupLimitExceeded,

			//efs.ErrCodeSecurityGroupNotFound for service response error code
			// "SecurityGroupNotFound".
			//
			// Returned if one of the specified security groups doesn't exist in the subnet's
			// VPC.
			efs.ErrCodeSecurityGroupNotFound,

			//efs.ErrCodeSubnetNotFound for service response error code
			// "SubnetNotFound".
			//
			// Returned if there is no subnet with ID SubnetId provided in the request.
			efs.ErrCodeSubnetNotFound,

			//efs.ErrCodeUnsupportedAvailabilityZone for service response error code
			// "UnsupportedAvailabilityZone".
			efs.ErrCodeUnsupportedAvailabilityZone,

			//efs.ErrCodeValidationException for service response error code
			// "ValidationException".
			//
			// Returned if the AWS Backup service is not available in the region that the
			// request was made.
			efs.ErrCodeValidationException:

			return apierror.New(apierror.ErrBadRequest, msg, aerr)
		case
			//efs.ErrCodeAccessPointLimitExceeded for service response error code
			// "AccessPointLimitExceeded".
			//
			// Returned if the AWS account has already created the maximum number of access
			// points allowed per file system.
			efs.ErrCodeAccessPointLimitExceeded,

			//efs.ErrCodeFileSystemLimitExceeded for service response error code
			// "FileSystemLimitExceeded".
			//
			// Returned if the AWS account has already created the maximum number of file
			// systems allowed per account.
			efs.ErrCodeFileSystemLimitExceeded,

			//efs.ErrCodeNetworkInterfaceLimitExceeded for service response error code
			// "NetworkInterfaceLimitExceeded".
			//
			// The calling account has reached the limit for elastic network interfaces
			// for the specific AWS Region. The client should try to delete some elastic
			// network interfaces or get the account limit raised. For more information,
			// see Amazon VPC Limits (https://docs.aws.amazon.com/AmazonVPC/latest/UserGuide/VPC_Appendix_Limits.html)
			// in the Amazon VPC User Guide (see the Network interfaces per VPC entry in
			// the table).
			efs.ErrCodeNetworkInterfaceLimitExceeded,

			//efs.ErrCodeNoFreeAddressesInSubnet for service response error code
			// "NoFreeAddressesInSubnet".
			//
			// Returned if IpAddress was not specified in the request and there are no free
			// IP addresses in the subnet.
			efs.ErrCodeNoFreeAddressesInSubnet,

			//efs.ErrCodeThroughputLimitExceeded for service response error code
			// "ThroughputLimitExceeded".
			//
			// Returned if the throughput mode or amount of provisioned throughput can't
			// be changed because the throughput limit of 1024 MiB/s has been reached.
			efs.ErrCodeThroughputLimitExceeded,

			//efs.ErrCodeTooManyRequests for service response error code
			// "TooManyRequests".
			//
			// Returned if you don’t wait at least 24 hours before changing the throughput
			// mode, or decreasing the Provisioned Throughput value.
			efs.ErrCodeTooManyRequests,

			// Limit Exceeded
			"LimitExceeded":

			return apierror.New(apierror.ErrLimitExceeded, msg, aerr)
		case
			//efs.ErrCodeDependencyTimeout for service response error code
			// "DependencyTimeout".
			//
			// The service timed out trying to fulfill the request, and the client should
			// try the call again.
			efs.ErrCodeDependencyTimeout,

			//efs.ErrCodeInternalServerError for service response error code
			// "InternalServerError".
			//
			// Returned if an error occurred on the server side.
			efs.ErrCodeInternalServerError,

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
