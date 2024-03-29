package api

import (
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awsutil"
	"github.com/aws/aws-sdk-go/service/efs"
	"github.com/aws/aws-sdk-go/service/iam"
	log "github.com/sirupsen/logrus"
)

// apiVersion is the API version
type apiVersion struct {
	// The version of the API
	Version string `json:"version"`
	// The git hash of the API
	GitHash string `json:"githash"`
	// The build timestamp of the API
	BuildStamp string `json:"buildstamp"`
}

// FileSystemCreateRequest is the request input for creating a filesystem
type FileSystemCreateRequest struct {
	// Name of the filesystem
	Name string

	// AccessPoints is an optional list of access points to create
	AccessPoints []*AccessPointCreateRequest

	// AccessPolicy is a set of flags to control access to the filesystem
	AccessPolicy *FileSystemAccessPolicy

	// BackupPolicy is the backup policy/status for the filesystem
	// Valid values are ENABLED | DISABLED
	BackupPolicy string

	// KMSKeyId used to encrypt the filesystem
	KmsKeyId string

	// After how long to transition to Infrequent Access storage
	// Valid values: NONE | AFTER_7_DAYS | AFTER_14_DAYS | AFTER_30_DAYS | AFTER_60_DAYS | AFTER_90_DAYS
	LifeCycleConfiguration string

	// Rule for transitioning back to the primary storage class from IA
	// Valid values: NONE | AFTER_1_ACCESS
	TransitionToPrimaryStorageClass string

	// OneZone creates the filesystem using the EFS OneZone storage classes
	OneZone bool

	// Security Group IDs to apply to the mount targets
	Sgs []string

	// subnets holds the list of subnets for one zone, not exposed to the client
	Subnets []string

	// Tags to apply to the filesystem
	Tags []*Tag
}

// FileSystemUpdateRequest is the input for updating a filesystem
type FileSystemUpdateRequest struct {
	// AccessPolicy is a set of flags to control access to the filesystem
	AccessPolicy *FileSystemAccessPolicy

	// BackupPolicy is the backup policy/status for the filesystem
	// Valid values are ENABLED | DISABLED
	BackupPolicy string

	// After how long to transition to Infrequent Access storage
	// Valid values: NONE | AFTER_7_DAYS | AFTER_14_DAYS | AFTER_30_DAYS | AFTER_60_DAYS | AFTER_90_DAYS
	LifeCycleConfiguration string

	// Rule for transitioning back to the primary storage class from IA
	// Valid values: NONE | AFTER_1_ACCESS
	TransitionToPrimaryStorageClass string

	// Tags to apply to the filesystem
	Tags []*Tag
}

// listFileSystemsResponse is the response for a list filesystems request
type listFileSystemsResponse []string

// FileSystemResponse represents a full filesystem service response
//
// A filesystem can have zero or more mount targets and zero or more access points.
type FileSystemResponse struct {
	// list of access points associated with the filesystem
	AccessPoints []*AccessPoint

	// AccessPolicy is a set of flags to control access to the filesystem
	AccessPolicy *FileSystemAccessPolicy

	// availability zone the filesystem is using
	AvailabilityZone string

	// BackupPolicy is the backup policy/status for the filesystem
	// Valid values are ENABLED | ENABLING | DISABLED | DISABLING
	BackupPolicy string

	// The time that the file system was created, in seconds (since 1970-01-01T00:00:00Z).
	CreationTime time.Time

	// The Amazon Resource Name (ARN) for the EFS file system, in the format arn:aws:elasticfilesystem:region:account-id:file-system/file-system-id
	FileSystemArn string

	// The ID of the file system, assigned by Amazon EFS.
	FileSystemId string

	// The ID of an KMS master key (CMK) used to encrypt the file system.
	KmsKeyId string

	// The lifecycle phase of the file system.
	LifeCycleState string

	// The lifecycle transition policy.
	// Valid values: NONE | AFTER_7_DAYS | AFTER_14_DAYS | AFTER_30_DAYS | AFTER_60_DAYS | AFTER_90_DAYS
	LifeCycleConfiguration string

	// Rule for transitioning back to the primary storage class from IA
	// Valid values: NONE | AFTER_1_ACCESS
	TransitionToPrimaryStorageClass string

	// A list of mount targets associated with the filesystem.
	MountTargets []*MountTarget

	// The name of the filesystem.
	Name string

	// The current number of access points that the file system has.
	NumberOfAccessPoints int64

	// The current number of mount targets that the file system has.
	NumberOfMountTargets int64

	// If true, the filesystem is using the EFS OneZone storage classes
	OneZone bool

	// The latest known metered size (in bytes) of data stored in the file system,
	// in its Value field, and the time at which that size was determined in its
	// Timestamp field. The Timestamp value is the integer number of seconds since
	// 1970-01-01T00:00:00Z. The SizeInBytes value doesn't represent the size of
	// a consistent snapshot of the file system, but it is eventually consistent
	// when there are no writes to the file system. That is, SizeInBytes represents
	// actual size only if the file system is not modified for a period longer than
	// a couple of hours. Otherwise, the value is not the exact size that the file
	// system was at any point in time.
	SizeInBytes *FileSystemSize

	// The tags associated with the file system.
	Tags []*Tag
}

// FileSystemAccessPolicy is a set of flags to control access to the filesystem
type FileSystemAccessPolicy struct {
	AllowAnonymousAccess      bool
	EnforceEncryptedTransport bool
	AllowEcsTaskExecutionRole bool
}

type FileSystemSize struct {
	// The time at which the size of data, returned in the Value field, was determined.
	// The value is the integer number of seconds since 1970-01-01T00:00:00Z.
	Timestamp time.Time

	// The latest known metered size (in bytes) of data stored in the file system.
	//
	// Value is a required field
	Value int64

	// The latest known metered size (in bytes) of data stored in the Infrequent
	// Access storage class.
	ValueInIA int64

	// The latest known metered size (in bytes) of data stored in the Standard storage
	// class.
	ValueInStandard int64
}

type MountTarget struct {
	// The unique and consistent identifier of the Availability Zone (AZ) that the
	// mount target resides in. For example, use1-az1 is an AZ ID for the us-east-1
	// Region and it has the same location in every AWS account.
	AvailabilityZoneId string

	// The name of the Availability Zone (AZ) that the mount target resides in.
	// AZs are independently mapped to names for each AWS account. For example,
	// the Availability Zone us-east-1a for your AWS account might not be the same
	// location as us-east-1a for another AWS account.
	AvailabilityZoneName string

	// Address at which the file system can be mounted by using the mount target.
	IpAddress string

	// Lifecycle state of the mount target.
	//
	// LifeCycleState is a required field
	LifeCycleState string

	// System-assigned mount target ID.
	//
	// MountTargetId is a required field
	MountTargetId string

	// The ID of the mount target's subnet.
	//
	// SubnetId is a required field
	SubnetId string
}

type AccessPoint struct {
	// The unique Amazon Resource Name (ARN) associated with the access point.
	AccessPointArn string

	// The ID of the access point, assigned by Amazon EFS.
	AccessPointId string

	// Identifies the lifecycle phase of the access point.
	LifeCycleState string

	// The name of the access point. This is the value of the Name tag.
	Name string

	// The full POSIX identity, including the user ID, group ID, and secondary group
	// IDs on the access point that is used for all file operations by NFS clients
	// using the access point.
	PosixUser *efs.PosixUser

	// The directory on the Amazon EFS file system that the access point exposes
	// as the root directory to NFS clients using the access point.
	RootDirectory *efs.RootDirectory
}

// FileSystemUserCreateRequest is the request payload for creating a filsystem user
type FileSystemUserCreateRequest struct {
	UserName string
}

// FileSystemUserResponse is the response payload for user operations
type FileSystemUserResponse struct {
	UserName          string
	AccessKeys        []*iam.AccessKeyMetadata `json:",omitempty"`
	AccessKey         *iam.AccessKey           `json:",omitempty"`
	DeletedAccessKeys []string                 `json:",omitempty"`
	Groups            []string                 `json:",omitempty"`
	Tags              []*Tag                   `json:",omitempty"`
}

// FileSystemUserUpdateRequest is the request payload for updating a user
type FileSystemUserUpdateRequest struct {
	ResetKey bool
}

// fileSystemFromEFS maps an EFS filesystem, list of moutn targets, and list of access points to a common struct
func fileSystemResponseFromEFS(fs *efs.FileSystemDescription, mts []*efs.MountTargetDescription, aps []*efs.AccessPointDescription, policy *FileSystemAccessPolicy, backup, ia, primary string) *FileSystemResponse {
	log.Debugf("mapping filesystem %s", awsutil.Prettify(fs))

	filesystem := FileSystemResponse{
		AccessPolicy:         policy,
		BackupPolicy:         backup,
		CreationTime:         aws.TimeValue(fs.CreationTime),
		FileSystemArn:        aws.StringValue(fs.FileSystemArn),
		FileSystemId:         aws.StringValue(fs.FileSystemId),
		KmsKeyId:             aws.StringValue(fs.KmsKeyId),
		LifeCycleState:       aws.StringValue(fs.LifeCycleState),
		Name:                 aws.StringValue(fs.Name),
		NumberOfMountTargets: aws.Int64Value(fs.NumberOfMountTargets),
	}

	if fs.AvailabilityZoneName != nil {
		filesystem.OneZone = true
		filesystem.AvailabilityZone = aws.StringValue(fs.AvailabilityZoneName)
	}

	tags := make([]*Tag, 0, len(fs.Tags))
	for _, t := range fs.Tags {
		tags = append(tags, &Tag{
			Key:   aws.StringValue(t.Key),
			Value: aws.StringValue(t.Value),
		})
	}
	filesystem.Tags = tags

	filesystem.NumberOfAccessPoints = int64(len(aps))
	accessPoints := make([]*AccessPoint, 0, len(aps))
	for _, a := range aps {
		log.Debugf("mapping accesspoint %s", awsutil.Prettify(a))
		accessPoints = append(accessPoints, &AccessPoint{
			AccessPointArn: aws.StringValue(a.AccessPointArn),
			AccessPointId:  aws.StringValue(a.AccessPointId),
			LifeCycleState: aws.StringValue(a.LifeCycleState),
			Name:           aws.StringValue(a.Name),
			PosixUser:      a.PosixUser,
			RootDirectory:  a.RootDirectory,
		})
	}
	filesystem.AccessPoints = accessPoints

	mountTargets := make([]*MountTarget, 0, len(mts))
	for _, m := range mts {
		log.Debugf("mapping mount target %s", awsutil.Prettify(m))
		mountTargets = append(mountTargets, &MountTarget{
			AvailabilityZoneId:   aws.StringValue(m.AvailabilityZoneId),
			AvailabilityZoneName: aws.StringValue(m.AvailabilityZoneName),
			IpAddress:            aws.StringValue(m.IpAddress),
			LifeCycleState:       aws.StringValue(m.LifeCycleState),
			MountTargetId:        aws.StringValue(m.MountTargetId),
			SubnetId:             aws.StringValue(m.SubnetId),
		})
	}
	filesystem.MountTargets = mountTargets

	if fs.SizeInBytes != nil {
		filesystem.SizeInBytes = &FileSystemSize{
			Timestamp:       aws.TimeValue(fs.SizeInBytes.Timestamp),
			Value:           aws.Int64Value(fs.SizeInBytes.Value),
			ValueInIA:       aws.Int64Value(fs.SizeInBytes.ValueInIA),
			ValueInStandard: aws.Int64Value(fs.SizeInBytes.ValueInStandard),
		}
	}

	filesystem.LifeCycleConfiguration = ia
	if filesystem.LifeCycleConfiguration == "" {
		filesystem.LifeCycleConfiguration = "NONE"
	}

	filesystem.TransitionToPrimaryStorageClass = primary
	if filesystem.TransitionToPrimaryStorageClass == "" {
		filesystem.TransitionToPrimaryStorageClass = "NONE"
	}

	return &filesystem
}

// AccessPointCreateRequest is the input for creating an access point
type AccessPointCreateRequest struct {
	Name string
	// https://docs.aws.amazon.com/sdk-for-go/api/service/efs/#PosixUser
	PosixUser *efs.PosixUser
	// https://docs.aws.amazon.com/sdk-for-go/api/service/efs/#CreationInfo
	RootDirectory *efs.RootDirectory
}

func accessPointResponseFromEFS(ap *efs.AccessPointDescription) *AccessPoint {
	return &AccessPoint{
		AccessPointArn: aws.StringValue(ap.AccessPointArn),
		AccessPointId:  aws.StringValue(ap.AccessPointId),
		LifeCycleState: aws.StringValue(ap.LifeCycleState),
		Name:           aws.StringValue(ap.Name),
		PosixUser:      ap.PosixUser,
		RootDirectory:  ap.RootDirectory,
	}
}

// filesystemUserResponseFromIAM maps IAM response to a common struct
func filesystemUserResponseFromIAM(u *iam.User, keys []*iam.AccessKeyMetadata) *FileSystemUserResponse {
	log.Debugf("mapping iam user %s", awsutil.Prettify(u))

	userName := aws.StringValue(u.UserName)

	// path is format: /spinup/%s/%s/%s/
	path := strings.Split(aws.StringValue(u.Path), "/")

	if len(path) > 2 {
		prefix := fmt.Sprintf("%s-", path[len(path)-2])

		log.Debugf("trimming prefix '%s' from username %s", prefix, userName)

		userName = strings.TrimPrefix(userName, prefix)
	}

	if keys == nil {
		keys = []*iam.AccessKeyMetadata{}
	}

	user := FileSystemUserResponse{
		AccessKeys: keys,
		UserName:   userName,
	}

	return &user
}
