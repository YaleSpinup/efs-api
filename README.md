# efs-api

This API provides simple restful API access to EFS services.

## Endpoints

```
GET /v1/efs/ping
GET /v1/efs/version
GET /v1/efs/metrics

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

DELETE `/v1/efs/{account}/filesystems/{group}/{id}`

| Response Code                 | Definition                               |
| ----------------------------- | -----------------------------------------|
| **202 Submitted**             | delete request is submitted              |
| **400 Bad Request**           | badly formed request                     |
| **404 Not Found**             | account or filesystem not found          |
| **409 Conflict**              | filesystem is not in the available state |
| **500 Internal Server Error** | a server error occurred                  |

## License

GNU Affero General Public License v3.0 (GNU AGPLv3)  
Copyright © 2020 Yale University
