# efs-api

This API provides simple restful API access to EFS services.

- [efs-api](#efs-api)
  - [Endpoints](#endpoints)
  - [Authentication](#authentication)
  - [Usage](#usage)
    - [Create a FileSystem](#create-a-filesystem)
      - [Example create request body](#example-create-request-body)
      - [Example create response body](#example-create-response-body)
    - [Update FileSystem](#update-filesystem)
      - [Example update request body](#example-update-request-body)
    - [List FileSystems](#list-filesystems)
      - [Example list response](#example-list-response)
    - [List FileSystems by group id](#list-filesystems-by-group-id)
      - [Example list by group response](#example-list-by-group-response)
    - [Get details about a FileSystem, including it's mount targets and access points](#get-details-about-a-filesystem-including-its-mount-targets-and-access-points)
      - [Example show response](#example-show-response)
    - [Delete a FileSystem and all associated mount targets and access points](#delete-a-filesystem-and-all-associated-mount-targets-and-access-points)
    - [Create an accesspoint for a filesystem](#create-an-accesspoint-for-a-filesystem)
      - [Example create accesspoint request](#example-create-accesspoint-request)
      - [Example create accesspoint response](#example-create-accesspoint-response)
    - [List accesspoints for a filesystem](#list-accesspoints-for-a-filesystem)
      - [Example list response](#example-list-response-1)
    - [Get details about an accesspoint](#get-details-about-an-accesspoint)
      - [Example get accesspoint response](#example-get-accesspoint-response)
    - [Delete an accesspoint](#delete-an-accesspoint)
      - [Example delete accesspoint response](#example-delete-accesspoint-response)
    - [Create a filesystem user](#create-a-filesystem-user)
      - [Example create user request](#example-create-user-request)
      - [Example create user response](#example-create-user-response)
    - [Update a filesystem user](#update-a-filesystem-user)
      - [Example update user request](#example-update-user-request)
      - [Example update user response](#example-update-user-response)
    - [List users for a filesystem](#list-users-for-a-filesystem)
      - [Example list users response](#example-list-users-response)
    - [Get details about a filesystem user](#get-details-about-a-filesystem-user)
      - [Example get user response](#example-get-user-response)
    - [Delete a filesystem user](#delete-a-filesystem-user)
      - [Example get user response](#example-get-user-response-1)
    - [Get task information for asynchronous tasks](#get-task-information-for-asynchronous-tasks)
      - [Example task response](#example-task-response)
  - [License](#license)

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
PUT    /v1/efs/{account}/filesystems/{group}/{id}
DELETE /v1/efs/{account}/filesystems/{group}/{id}

POST   /v1/efs/{account}/filesystems/{group}/{id}/aps
GET    /v1/efs/{account}/filesystems/{group}/{id}/aps
PUT    /v1/efs/{account}/filesystems/{group}/{id}/aps/{apid}
DELETE /v1/efs/{account}/filesystems/{group}/{id}/aps/{apid}
```

## Authentication

Authentication is accomplished via a pre-shared key.  This is done via the `X-Auth-Token` header.

## Usage

### Create a FileSystem

Creating a filesystem generates an EFS filesystem, and mount targets in all of the configured subnets
with the passed security groups.  If no security groups are passed, the default will be used.  If OneZone
is set to true, the filesystem will be set to use the "EFS OneZone" storage class and a subnet/az will be
chosen at random.

Create requests are asynchronous and returns a task ID in the header `X-Flywheel-Task`.  This header can
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
    "LifeCycleConfiguration": "NONE | AFTER_7_DAYS | AFTER_14_DAYS | AFTER_30_DAYS | AFTER_60_DAYS | AFTER_90_DAYS",
    "TransitionToPrimaryStorageClass": "NONE | AFTER_1_ACCESS",
    "BackupPolicy": "ENABLED | DISABLED",
    "OneZone": true,
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
    "AvailabilityZone": "us-east-1a",
    "BackupPolicy": "ENABLED | DISABLED",
    "CreationTime": "2020-08-06T11:14:45Z",
    "FileSystemArn": "arn:aws:elasticfilesystem:us-east-1:1234567890:file-system/fs-9876543",
    "FileSystemId": "fs-9876543",
    "KmsKeyId": "arn:aws:kms:us-east-1:1234567890:key/0000000-1111-1111-1111-33333333333",
    "LifeCycleState": "creating",
    "LifeCycleConfiguration": "NONE | AFTER_7_DAYS | AFTER_14_DAYS | AFTER_30_DAYS | AFTER_60_DAYS | AFTER_90_DAYS",
    "TransitionToPrimaryStorageClass": "NONE | AFTER_1_ACCESS",
    "MountTargets": [],
    "Name": "myAwesomeFilesystem",
    "NumberOfAccessPoints": 0,
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

### Update FileSystem

The update endpoint allows for updating Tags, BackupPolicy, LifeCycleConfiguration and TransitionToPrimaryStorageClass
for the filesystem.  All fields are optional.  Tags are additive (ie. existing tags are not removed), existing tags will be updated.

Update requests are asynchronous and returns a task ID in the header `X-Flywheel-Task`.  This header can
be used to get the task information and logs from the flywheel HTTP endpoint.

PUT `/v1/efs/{account}/filesystems/{group}/{id}`

| Response Code                 | Definition                      |
| ----------------------------- | --------------------------------|
| **200 OK**                    | create a filesystem             |
| **400 Bad Request**           | badly formed request            |
| **404 Not Found**             | account not found               |
| **500 Internal Server Error** | a server error occurred         |

#### Example update request body

```json
{
    "LifeCycleConfiguration": "NONE | AFTER_7_DAYS | AFTER_14_DAYS | AFTER_30_DAYS | AFTER_60_DAYS | AFTER_90_DAYS",
    "TransitionToPrimaryStorageClass": "NONE | AFTER_1_ACCESS",
    "BackupPolicy": "ENABLED | DISABLED",
    "Tags": [
        {
            "Key": "Bill.Me",
            "Value": "Now"
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
    "BackupPolicy": "ENABLED | ENABLING | DISABLED | DISABLING",
    "CreationTime": "2020-08-06T11:14:45Z",
    "FileSystemArn": "arn:aws:elasticfilesystem:us-east-1:1234567890:file-system/fs-9876543",
    "FileSystemId": "fs-9876543",
    "KmsKeyId": "arn:aws:kms:us-east-1:1234567890:key/0000000-1111-1111-1111-33333333333",
    "LifeCycleState": "available",
    "LifeCycleConfiguration": "NONE | AFTER_7_DAYS | AFTER_14_DAYS | AFTER_30_DAYS | AFTER_60_DAYS | AFTER_90_DAYS",
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
    "NumberOfAccessPoints": 0,
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

### Create an accesspoint for a filesystem

Creating an accesspoint generates an accesspoint for a filesystem.

POST `/v1/efs/{account}/filesystems/{group}/{id}/aps`

#### Example create accesspoint request

The request takes a posix user and a root directory.  Both are optional.  Setting the Posix User allows overriding the UID and
GID of anyone mounting the accesspoint.  This is useful for cases where the filesystem is accessed by a non-root user such as
in a container.  Setting the root directly will override the root directory of the filesystem when mounting via the accesspoint.

[PosixUser](https://docs.aws.amazon.com/sdk-for-go/api/service/efs/#PosixUser)
[RootDirectory](https://docs.aws.amazon.com/sdk-for-go/api/service/efs/#RootDirectory)]

```json
    "Name": "ap1",
    "PosixUser": {
        "Uid": 1000,
        "Gid": 1000,
    },
    "RootDirectory": {
        "Path": "/somedir",
    }
```

#### Example create accesspoint response

```json
```

| Response Code                 | Definition                      |
| ----------------------------- | --------------------------------|
| **200 OK**                    | create an access point          |
| **400 Bad Request**           | badly formed request            |
| **404 Not Found**             | account not found               |
| **500 Internal Server Error** | a server error occurred         |

### List accesspoints for a filesystem

GET `/v1/efs/{account}/filesystems/{group}/{id}/aps`

#### Example list response

```json
[
    "fsap-9876543",
    "fsap-abcdefg"
]
```

| Response Code                 | Definition                      |
| ----------------------------- | --------------------------------|
| **200 OK**                    | return the list of accesspoints |
| **400 Bad Request**           | badly formed request            |
| **404 Not Found**             | account not found               |
| **500 Internal Server Error** | a server error occurred         |

### Get details about an accesspoint

GET `/v1/efs/{account}/filesystems/{group}/{id}/aps/{apid}`

#### Example get accesspoint response

```json
{
    "AccessPointArn": "arn:aws:elasticfilesystem:us-east-1:012345678910:access-point/fsap-0e84a50717caf79a6",
    "AccessPointId": "fsap-0e84a50717caf79a6",
    "LifeCycleState": "creating",
    "Name": "myAwesomeFilesystem12-ap1",
    "PosixUser": {
        "Gid": 1000,
        "SecondaryGids": null,
        "Uid": 1000
    },
    "RootDirectory": {
        "CreationInfo": null,
        "Path": "/somedir"
    }
}
```

| Response Code                 | Definition                        |
| ----------------------------- | ----------------------------------|
| **200 OK**                    | return details of an accesspoint  |
| **400 Bad Request**           | badly formed request              |
| **404 Not Found**             | account, fs, or ap not found      |
| **500 Internal Server Error** | a server error occurred           |

### Delete an accesspoint

DELETE `/v1/efs/{account}/filesystems/{group}/{id}/aps/{apid}`

#### Example delete accesspoint response

```json
OK
```

### Create a filesystem user

Creates a user with full access to a filesystem

POST `/v1/efs/{account}/filesystems/{group}/{id}/users`

#### Example create user request

```json
{
    "Username": "someuser"
}
```

#### Example create user response

```json
{
    "UserName": "someuser",
}
```

| Response Code                 | Definition              |
| ----------------------------- | ------------------------|
| **200 OK**                    | create a user           |
| **400 Bad Request**           | badly formed request    |
| **404 Not Found**             | account not found       |
| **500 Internal Server Error** | a server error occurred |

### Update a filesystem user

Updating a user is primarily used to reset the access keys for that user.

PUT `/v1/efs/{account}/filesystems/{group}/{id}/users/{username}`

#### Example update user request

```json
{
    "ResetKey": true,
}
```

#### Example update user response

```json
{
    "UserName": "someuser",
    "AccessKey": {
        "AccessKeyId": "XXXXXXXXXX",
        "CreateDate": "2021-10-19T22:58:03Z",
        "SecretAccessKey": "yyyyyyyyyyyyyyyyyyyyyyyyyy",
        "Status": "Active",
        "UserName": "myAwesomeFilesystem-someuser"
    },
    "DeletedAccessKeys": [
        "ZZZZZZZZZZZ"
    ]
}
```

| Response Code                 | Definition              |
| ----------------------------- | ------------------------|
| **200 OK**                    | update a user           |
| **400 Bad Request**           | badly formed request    |
| **404 Not Found**             | account not found       |
| **500 Internal Server Error** | a server error occurred |

### List users for a filesystem

GET ``/v1/efs/{account}/filesystems/{group}/{id}/users`

#### Example list users response

```json
[
    "someuser",
    "someotheruser"
]
```

| Response Code                 | Definition              |
| ----------------------------- | ------------------------|
| **200 OK**                    | list all users          |
| **400 Bad Request**           | badly formed request    |
| **404 Not Found**             | account not found       |
| **500 Internal Server Error** | a server error occurred |

### Get details about a filesystem user

GET ``/v1/efs/{account}/filesystems/{group}/{id}/users/{username}`

#### Example get user response

```json
{
    "UserName": "someuser",
}
```

| Response Code                 | Definition              |
| ----------------------------- | ------------------------|
| **200 OK**                    | get a user              |
| **400 Bad Request**           | badly formed request    |
| **404 Not Found**             | account not found       |
| **500 Internal Server Error** | a server error occurred |

### Delete a filesystem user

DELETE ``/v1/efs/{account}/filesystems/{group}/{id}/users/{username}`

#### Example get user response

```json
OK
```

| Response Code                 | Definition              |
| ----------------------------- | ------------------------|
| **200 OK**                    | delete a user           |
| **400 Bad Request**           | badly formed request    |
| **404 Not Found**             | account not found       |
| **500 Internal Server Error** | a server error occurred |

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
Copyright © 2021 Yale University
