# efs-api

This API provides simple restful API access to EFS services.

## Endpoints

```
GET /v1/efs/ping
GET /v1/efs/version
GET /v1/efs/metrics

GET /v1/efs/flywheel?task=xxx[&task=yyy&task=zzz]

GET    /v1/efs/{account}/filesystems
GET    /v1/efs/{account}/filesystems/{group}
POST   /v1/efs/{account}/filesystems/{group}
GET    /v1/efs/{account}/filesystems/{group}/{id}
DELETE /v1/efs/{account}/filesystems/{group}/{id}
```

## Authentication

Authentication is accomplished via a pre-shared key.  This is done via the `X-Auth-Token` header.

## Usage

### Create a FileSystem

Creating a filesystem generates an EFS filesystem, and mount targets in all of the configured subnets
with the passed security groups.  If no security groups are passed, the default will be used.

Create requests are asynchronous and returns a task ID in the header `X-Spinup-Task`.  This header can
be used to get the task information and logs from the flywheel HTTP endpoint.

POST `/v1/efs/{account}/filesystems/{group}`

| Response Code                 | Definition                      |
| ----------------------------- | --------------------------------|
| **200 OK**                    | create a filesystem             |
| **400 Bad Request**           | badly formed request            |
| **404 Not Found**             | account not found               |
| **500 Internal Server Error** | a server error occurred         |

#### Example create request body

```json
{
    "Name": "myAwesomeFilesystem",
    "KmsKeyId": "arn:aws:kms:us-east-1:1234567890:key/0000000-1111-1111-1111-33333333333",
    "Sgs": ["sg-abc123456789"],
    "Tags": [
        {
            "Key": "Bill.Me",
            "Value": "Later"
        }
    ]
}
```

#### Example create response body

```json
{
    "AccessPoints": [],
    "CreationTime": "2020-08-06T11:14:45Z",
    "FileSystemArn": "arn:aws:elasticfilesystem:us-east-1:1234567890:file-system/fs-9876543",
    "FileSystemId": "fs-9876543",
    "KmsKeyId": "arn:aws:kms:us-east-1:1234567890:key/0000000-1111-1111-1111-33333333333",
    "LifeCycleState": "creating",
    "MountTargets": [
        {
            "AvailabilityZoneId": "use1-az2",
            "AvailabilityZoneName": "us-east-1a",
            "IpAddress": "10.1.2.111",
            "LifeCycleState": "creating",
            "MountTargetId": "fsmt-1111111",
            "SubnetId": "subnet-MjIyMjIyMjIyMjIyMjI"
        },
        {
            "AvailabilityZoneId": "use1-az1",
            "AvailabilityZoneName": "us-east-1d",
            "IpAddress": "10.1.3.111",
            "LifeCycleState": "creating",
            "MountTargetId": "fsmt-2222222",
            "SubnetId": "subnet-MzMzMzMzMzMzMzMzMzM"
        }
    ],
    "Name": "myAwesomeFilesystem",
    "NumberOfMountTargets": 0,
    "SizeInBytes": {
        "Timestamp": "0001-01-01T00:00:00Z",
        "Value": 0,
        "ValueInIA": 0,
        "ValueInStandard": 0
    },
    "Tags": [
        {
            "Key": "Name",
            "Value": "myAwesomeFilesystem"
        },
        {
            "Key": "spinup:org",
            "Value": "spindev"
        },
        {
            "Key": "spinup:spaceid",
            "Value": "spindev-00001"
        },
        {
            "Key": "Bill.Me",
            "Value": "Later"
        }
    ]
}
```

### List FileSystems

GET `/v1/efs/{account}/filesystems`

| Response Code                 | Definition                      |
| ----------------------------- | --------------------------------|
| **200 OK**                    | return the list of filesystems  |
| **400 Bad Request**           | badly formed request            |
| **404 Not Found**             | account not found               |
| **500 Internal Server Error** | a server error occurred         |

#### Example list response

```json
[
    "spindev-00001/fs-1234567",
    "spindev-00001/fs-7654321",
    "spindev-00002/fs-9876543",
    "spindev-00003/fs-abcdefg"
]
```

### List FileSystems by group id

GET `/v1/efs/{account}/filesystems/{group}`

| Response Code                 | Definition                      |
| ----------------------------- | --------------------------------|
| **200 OK**                    | return the list of filesystems  |
| **400 Bad Request**           | badly formed request            |
| **404 Not Found**             | account not found               |
| **500 Internal Server Error** | a server error occurred         |

#### Example list by group response

```json
[
    "fs-9876543",
    "fs-abcdefg"
]
```

### Get details about a FileSystem, including it's mount targets and access points

GET `/v1/efs/{account}/filesystems/{group}/{id}`

| Response Code                 | Definition                      |
| ----------------------------- | --------------------------------|
| **200 OK**                    | return details of a filesystem  |
| **400 Bad Request**           | badly formed request            |
| **404 Not Found**             | account or filesystem not found |
| **500 Internal Server Error** | a server error occurred         |

#### Example show response

```json
{
    "AccessPoints": [],
    "CreationTime": "2020-08-06T11:14:45Z",
    "FileSystemArn": "arn:aws:elasticfilesystem:us-east-1:1234567890:file-system/fs-9876543",
    "FileSystemId": "fs-9876543",
    "KmsKeyId": "arn:aws:kms:us-east-1:1234567890:key/0000000-1111-1111-1111-33333333333",
    "LifeCycleState": "available",
    "MountTargets": [
        {
            "AvailabilityZoneId": "use1-az2",
            "AvailabilityZoneName": "us-east-1a",
            "IpAddress": "10.1.2.111",
            "LifeCycleState": "available",
            "MountTargetId": "fsmt-1111111",
            "SubnetId": "subnet-MjIyMjIyMjIyMjIyMjI"
        },
        {
            "AvailabilityZoneId": "use1-az1",
            "AvailabilityZoneName": "us-east-1d",
            "IpAddress": "10.1.3.111",
            "LifeCycleState": "available",
            "MountTargetId": "fsmt-2222222",
            "SubnetId": "subnet-MzMzMzMzMzMzMzMzMzM"
        }
    ],
    "Name": "myAwesomeFilesystem",
    "NumberOfMountTargets": 2,
    "SizeInBytes": {
        "Timestamp": "0001-01-01T00:00:00Z",
        "Value": 0,
        "ValueInIA": 0,
        "ValueInStandard": 0
    },
    "Tags": [
        {
            "Key": "Name",
            "Value": "myAwesomeFilesystem"
        },
        {
            "Key": "spinup:org",
            "Value": "spindev"
        },
        {
            "Key": "spinup:spaceid",
            "Value": "spindev-00001"
        },
        {
            "Key": "Bill.Me",
            "Value": "Later"
        }
    ]
}
```

### Delete a FileSystem and all associated mount targets and access points

Delete requests are asynchronous and returns a task ID in the header `X-Spinup-Task`.  This header can
be used to get the task information and logs from the flywheel HTTP endpoint.

DELETE `/v1/efs/{account}/filesystems/{group}/{id}`

| Response Code                 | Definition                               |
| ----------------------------- | -----------------------------------------|
| **202 Submitted**             | delete request is submitted              |
| **400 Bad Request**           | badly formed request                     |
| **404 Not Found**             | account or filesystem not found          |
| **409 Conflict**              | filesystem is not in the available state |
| **500 Internal Server Error** | a server error occurred                  |


### Get task information for asynchronous tasks

GET /v1/efs/flywheel?task=xxx[&task=yyy&task=zzz]

#### Example task response

```json
{
    "a4b82c65-1ec7-4744-8098-70b03bd0f91d": {
        "checkin_at": "2020-08-21T12:21:56.9060761Z",
        "completed_at": "2020-08-21T12:21:56.9765756Z",
        "created_at": "2020-08-21T12:21:40.6471312Z",
        "id": "a4b82c65-1ec7-4744-8098-70b03bd0f91d",
        "status": "completed",
        "events": [
            "2020-08-21T12:21:40.6643552Z starting task a4b82c65-1ec7-4744-8098-70b03bd0f91d",
            "2020-08-21T12:21:40.7473309Z 2020-08-21T12:21:40.7396794Z checkin task a4b82c65-1ec7-4744-8098-70b03bd0f91d",
            "2020-08-21T12:21:40.7498547Z listed mount targets for filesystem fs-528f5fd0",
            "2020-08-21T12:21:40.7594181Z 2020-08-21T12:21:40.752216Z checkin task a4b82c65-1ec7-4744-8098-70b03bd0f91d",
            "2020-08-21T12:21:47.6276397Z waiting for number of mount targets for filesystem fs-528f5fd0 to be 0",
            "2020-08-21T12:21:56.5784279Z 2020-08-21T12:21:56.5727943Z checkin task a4b82c65-1ec7-4744-8098-70b03bd0f91d",
            "2020-08-21T12:21:56.5793964Z waiting for number of mount targets for filesystem fs-528f5fd0 to be 0",
            "2020-08-21T12:21:56.9091114Z 2020-08-21T12:21:56.9060761Z checkin task a4b82c65-1ec7-4744-8098-70b03bd0f91d",
            "2020-08-21T12:21:56.912431Z deleting filesystem fs-528f5fd0",
            "2020-08-21T12:21:56.9781826Z 2020-08-21T12:21:56.9765756Z complete task a4b82c65-1ec7-4744-8098-70b03bd0f91d"
        ]
    },
    "f6ca4ff3-81a1-480b-a1c3-e02ae93a28df": {
        "checkin_at": "2020-08-21T12:23:07.6148118Z",
        "completed_at": "2020-08-21T12:23:07.6872874Z",
        "created_at": "2020-08-21T12:22:38.2393195Z",
        "id": "f6ca4ff3-81a1-480b-a1c3-e02ae93a28df",
        "status": "completed",
        "events": [
            "2020-08-21T12:22:38.240863Z starting task f6ca4ff3-81a1-480b-a1c3-e02ae93a28df",
            "2020-08-21T12:22:38.3512033Z 2020-08-21T12:22:38.3489356Z checkin task f6ca4ff3-81a1-480b-a1c3-e02ae93a28df",
            "2020-08-21T12:22:38.3519238Z listed mount targets for filesystem fs-d8b2625a",
            "2020-08-21T12:22:48.3812638Z 2020-08-21T12:22:48.3759051Z checkin task f6ca4ff3-81a1-480b-a1c3-e02ae93a28df",
            "2020-08-21T12:22:48.3824365Z waiting for number of mount targets for filesystem fs-d8b2625a to be 0",
            "2020-08-21T12:23:07.2435381Z 2020-08-21T12:23:07.2405341Z checkin task f6ca4ff3-81a1-480b-a1c3-e02ae93a28df",
            "2020-08-21T12:23:07.246537Z waiting for number of mount targets for filesystem fs-d8b2625a to be 0",
            "2020-08-21T12:23:07.6224481Z 2020-08-21T12:23:07.6148118Z checkin task f6ca4ff3-81a1-480b-a1c3-e02ae93a28df",
            "2020-08-21T12:23:07.6232738Z deleting filesystem fs-d8b2625a",
            "2020-08-21T12:23:07.6911809Z 2020-08-21T12:23:07.6872874Z complete task f6ca4ff3-81a1-480b-a1c3-e02ae93a28df"
        ]
    }
}
```

## License

GNU Affero General Public License v3.0 (GNU AGPLv3)  
Copyright © 2020 Yale University
