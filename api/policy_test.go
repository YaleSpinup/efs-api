package api

import (
	"reflect"
	"testing"

	"github.com/YaleSpinup/aws-go/services/iam"
)

func Test_orgTagAccessPolicy(t *testing.T) {
	type args struct {
		org string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "org policy",
			args: args{
				org: "testOrg",
			},
			want: `{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Action":["*"],"Resource":["*"],"Condition":{"StringEquals":{"aws:ResourceTag/spinup:org":["testOrg"]}}}]}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := orgTagAccessPolicy(tt.args.org)
			if (err != nil) != tt.wantErr {
				t.Errorf("orgTagAccessPolicy() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("orgTagAccessPolicy() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_server_filesystemUserCreatePolicy(t *testing.T) {
	type fields struct {
		org string
	}
	tests := []struct {
		name    string
		fields  fields
		want    string
		wantErr bool
	}{
		{
			name: "test org",
			fields: fields{
				org: "testOrg",
			},
			want: `{"Version":"2012-10-17","Statement":[{"Sid":"CreateRepositoryUser","Effect":"Allow","Action":["iam:CreatePolicy","iam:UntagUser","iam:GetPolicyVersion","iam:AddUserToGroup","iam:GetPolicy","iam:ListAttachedGroupPolicies","iam:ListGroupPolicies","iam:AttachGroupPolicy","iam:GetUser","iam:CreatePolicyVersion","iam:CreateUser","iam:GetGroup","iam:CreateGroup","iam:TagUser"],"Resource":["arn:aws:iam::*:group/*","arn:aws:iam::*:policy/spinup/testOrg/*","arn:aws:iam::*:user/spinup/testOrg/*"]},{"Sid":"ListRepositoryUserPolicies","Effect":"Allow","Action":["iam:ListPolicies"],"Resource":["*"]}]}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &server{
				org: tt.fields.org,
			}
			got, err := s.filesystemUserCreatePolicy()
			if (err != nil) != tt.wantErr {
				t.Errorf("server.filesystemUserCreatePolicy() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("server.filesystemUserCreatePolicy() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_server_filesystemUserDeletePolicy(t *testing.T) {
	type fields struct {
		org string
	}
	tests := []struct {
		name    string
		fields  fields
		want    string
		wantErr bool
	}{
		{
			name: "test org",
			fields: fields{
				org: "testOrg",
			},
			want: `{"Version":"2012-10-17","Statement":[{"Sid":"DeleteRepositoryUser","Effect":"Allow","Action":["iam:DeleteAccessKey","iam:RemoveUserFromGroup","iam:ListAccessKeys","iam:ListGroupsForUser","iam:ListUsers","iam:DeleteUser","iam:GetUser"],"Resource":["arn:aws:iam::*:user/spinup/testOrg/*","arn:aws:iam::*:group/spinup/testOrg/*"]}]}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &server{
				org: tt.fields.org,
			}
			got, err := s.filesystemUserDeletePolicy()
			if (err != nil) != tt.wantErr {
				t.Errorf("server.filesystemUserDeletePolicy() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("server.filesystemUserDeletePolicy() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_server_filesystemUserUpdatePolicy(t *testing.T) {
	type fields struct {
		org string
	}
	tests := []struct {
		name    string
		fields  fields
		want    string
		wantErr bool
	}{
		{
			name: "test org",
			fields: fields{
				org: "testOrg",
			},
			want: `{"Version":"2012-10-17","Statement":[{"Sid":"UpdateRepositoryUser","Effect":"Allow","Action":["iam:GetUser","iam:UntagUser","iam:DeleteAccessKey","iam:RemoveUserFromGroup","iam:TagUser","iam:CreateAccessKey","iam:ListAccessKeys"],"Resource":["arn:aws:iam::*:user/spinup/testOrg/*","arn:aws:iam::*:group/spinup/testOrg/SpinupEFSAdminGroup-testOrg"]}]}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &server{
				org: tt.fields.org,
			}
			got, err := s.filesystemUserUpdatePolicy()
			if (err != nil) != tt.wantErr {
				t.Errorf("server.filesystemUserUpdatePolicy() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("server.filesystemUserUpdatePolicy() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_efsPolicyFromFilSystemAccessPolicy(t *testing.T) {
	type args struct {
		account string
		group   string
		fsArn   string
		policy  *FileSystemAccessPolicy
	}
	tests := []struct {
		name string
		args args
		want *iam.PolicyDocument
	}{
		{
			name: "empty policy",
			args: args{},
			want: nil,
		},
		{
			name: "DenyUnencryptedTransport",
			args: args{
				account: "1234567890",
				group:   "mygroup",
				fsArn:   "arn:aws:elasticfilesystem:us-east-1:1234567890:file-system/fs-aabbccddeeff",
				policy:  &FileSystemAccessPolicy{true, true, false},
			},
			want: &iam.PolicyDocument{
				Version: "2012-10-17",
				Id:      "efs-resource-policy-document",
				Statement: []iam.StatementEntry{
					{
						Sid:       "DenyUnencryptedTransport",
						Effect:    "Deny",
						Principal: iam.Principal{"AWS": []string{"*"}},
						Action:    []string{"*"},
						Resource:  []string{"arn:aws:elasticfilesystem:us-east-1:1234567890:file-system/fs-aabbccddeeff"},
						Condition: iam.Condition{
							"Bool": iam.ConditionStatement{
								"aws:SecureTransport": []string{"false"},
							},
						},
					},
				},
			},
		},
		{
			name: "AllowECSAccessFromHomeSpace",
			args: args{
				account: "1234567890",
				group:   "mygroup",
				fsArn:   "arn:aws:elasticfilesystem:us-east-1:1234567890:file-system/fs-aabbccddeeff",
				policy:  &FileSystemAccessPolicy{true, false, true},
			},
			want: &iam.PolicyDocument{
				Version: "2012-10-17",
				Id:      "efs-resource-policy-document",
				Statement: []iam.StatementEntry{
					{
						Sid:    "AllowECSAccessFromHomeSpace",
						Effect: "Allow",
						Principal: iam.Principal{
							"AWS": []string{"arn:aws:iam::1234567890:role/mygroup-ecsTaskExecution"},
						},
						Action: []string{
							"elasticfilesystem:ClientRootAccess",
							"elasticfilesystem:ClientWrite",
							"elasticfilesystem:ClientMount",
						},
						Resource: []string{"arn:aws:elasticfilesystem:us-east-1:1234567890:file-system/fs-aabbccddeeff"},
						Condition: iam.Condition{
							"Bool": iam.ConditionStatement{
								"elasticfilesystem:AccessedViaMountTarget": []string{"true"},
							},
						},
					},
				},
			},
		},
		{
			name: "disable anonymous access",
			args: args{
				account: "1234567890",
				group:   "mygroup",
				fsArn:   "arn:aws:elasticfilesystem:us-east-1:1234567890:file-system/fs-aabbccddeeff",
				policy:  &FileSystemAccessPolicy{false, false, false},
			},
			want: &iam.PolicyDocument{
				Version: "2012-10-17",
				Id:      "efs-resource-policy-document",
				Statement: []iam.StatementEntry{
					{
						Sid:       "DenyAnonymousAccess",
						Effect:    "Allow",
						Principal: iam.Principal{"AWS": []string{"*"}},
						Action: []string{
							"elasticfilesystem:ClientRootAccess",
							"elasticfilesystem:ClientWrite",
						},
						Resource: []string{"arn:aws:elasticfilesystem:us-east-1:1234567890:file-system/fs-aabbccddeeff"},
						Condition: iam.Condition{
							"Bool": iam.ConditionStatement{
								"elasticfilesystem:AccessedViaMountTarget": []string{"true"},
							},
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := efsPolicyFromFilSystemAccessPolicy(tt.args.account, tt.args.group, tt.args.fsArn, tt.args.policy); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("efsPolicyFromFilSystemAccessPolicy() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_filSystemAccessPolicyFromEfsPolicy(t *testing.T) {
	type args struct {
		policy string
	}
	tests := []struct {
		name    string
		args    args
		want    *FileSystemAccessPolicy
		wantErr bool
	}{
		{
			name: "empty",
			want: nil,
		},
		{
			name: "defaults",
			args: args{
				policy: `{"Statement": []}`,
			},
			want: &FileSystemAccessPolicy{true, false, false},
		},
		{
			name: "DenyUnencryptedTransport",
			args: args{
				policy: `{"Statement": [{"Sid":"DenyUnencryptedTransport"}]}`,
			},
			want: &FileSystemAccessPolicy{true, true, false},
		},
		{
			name: "DenyAnonymousAccess",
			args: args{
				policy: `{"Statement": [{"Sid":"DenyAnonymousAccess"}]}`,
			},
			want: &FileSystemAccessPolicy{false, false, false},
		},
		{
			name: "AllowECSAccessFromHomeSpace",
			args: args{
				policy: `{"Statement": [{"Sid":"AllowECSAccessFromHomeSpace"}]}`,
			},
			want: &FileSystemAccessPolicy{true, false, true},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := filSystemAccessPolicyFromEfsPolicy(tt.args.policy)
			if (err != nil) != tt.wantErr {
				t.Errorf("filSystemAccessPolicyFromEfsPolicy() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("filSystemAccessPolicyFromEfsPolicy() = %v, want %v", got, tt.want)
			}
		})
	}
}
