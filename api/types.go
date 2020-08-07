package api

import (
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awsutil"
	"github.com/aws/aws-sdk-go/service/efs"
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

// listFileSystemsResponse is the response for a list filesystems request
type listFileSystemsResponse []string

// FileSystemCreateRequest is the request input for creating a filesystem
type FileSystemCreateRequest struct {
	// Name of the filesystem
	Name string
	// KMSKeyId used to encrypt the filesystem
	KmsKeyId string
	// Tags to apply to the filesystem
	Tags []*Tag
}

// FileSystemResponse represents a full filesystem service response
//
// A filesystem can have zero or more mount targets and zero or more access points.
type FileSystemResponse struct {
	// list of access points associated with the filesystem
	AccessPoints []*AccessPoint

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

	// A list of mount targets associated with the filesystem.
	MountTargets []*MountTarget

	// The name of the filesystem.
	Name string

	// The current number of mount targets that the file system has.
	NumberOfMountTargets int64

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
	PosixUser *PosixUser

	// The directory on the Amazon EFS file system that the access point exposes
	// as the root directory to NFS clients using the access point.
	RootDirectory *RootDirectory
}

type PosixUser struct{}
type RootDirectory struct{}

type Tag struct {
	Key   string
	Value string
}

// fileSystemFromEFS maps an EFS filesystem, list of moutn targets, and list of access points to a common struct
func fileSystemResponseFromEFS(fs *efs.FileSystemDescription, mts []*efs.MountTargetDescription, aps []*efs.AccessPointDescription) *FileSystemResponse {
	log.Debugf("mapping filesystem %s", awsutil.Prettify(fs))

	filesystem := FileSystemResponse{
		CreationTime:         aws.TimeValue(fs.CreationTime),
		FileSystemArn:        aws.StringValue(fs.FileSystemArn),
		FileSystemId:         aws.StringValue(fs.FileSystemId),
		KmsKeyId:             aws.StringValue(fs.KmsKeyId),
		LifeCycleState:       aws.StringValue(fs.LifeCycleState),
		Name:                 aws.StringValue(fs.Name),
		NumberOfMountTargets: aws.Int64Value(fs.NumberOfMountTargets),
	}

	tags := make([]*Tag, 0, len(fs.Tags))
	for _, t := range fs.Tags {
		tags = append(tags, &Tag{
			Key:   aws.StringValue(t.Key),
			Value: aws.StringValue(t.Value),
		})
	}
	filesystem.Tags = tags

	accessPoints := make([]*AccessPoint, 0, len(aps))
	for _, a := range aps {
		log.Debugf("mapping accesspoint %s", awsutil.Prettify(a))
		accessPoints = append(accessPoints, &AccessPoint{
			AccessPointArn: aws.StringValue(a.AccessPointArn),
			AccessPointId:  aws.StringValue(a.AccessPointId),
			LifeCycleState: aws.StringValue(a.LifeCycleState),
			Name:           aws.StringValue(a.Name),
			PosixUser:      &PosixUser{},
			RootDirectory:  &RootDirectory{},
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

	return &filesystem
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
