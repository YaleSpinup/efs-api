package api

import (
	"testing"
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
			want: `{"Version":"2012-10-17","Statement":[{"Sid":"DeleteRepositoryUser","Effect":"Allow","Action":["iam:DeleteAccessKey","iam:RemoveUserFromGroup","iam:ListAccessKeys","iam:ListGroupsForUser","iam:DeleteUser","iam:GetUser"],"Resource":["arn:aws:iam::*:user/spinup/testOrg/*"]}]}`,
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
			want: `{"Version":"2012-10-17","Statement":[{"Sid":"UpdateRepositoryUser","Effect":"Allow","Action":["iam:UntagUser","iam:DeleteAccessKey","iam:RemoveUserFromGroup","iam:TagUser","iam:CreateAccessKey","iam:ListAccessKeys"],"Resource":["arn:aws:iam::*:user/spinup/testOrg/*"]}]}`,
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
