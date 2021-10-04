package api

import (
	"encoding/json"
	"fmt"

	"github.com/YaleSpinup/aws-go/services/iam"
	log "github.com/sirupsen/logrus"
)

// orgTagAccessPolicy generates the org tag conditional policy to be passed inline when assuming a role
func orgTagAccessPolicy(org string) (string, error) {
	log.Debugf("generating org policy document")

	policy := iam.PolicyDocument{
		Version: "2012-10-17",
		Statement: []iam.StatementEntry{
			{
				Effect:   "Allow",
				Action:   []string{"*"},
				Resource: []string{"*"},
				Condition: iam.Condition{
					"StringEquals": iam.ConditionStatement{
						"aws:ResourceTag/spinup:org": []string{org},
					},
				},
			},
		},
	}

	j, err := json.Marshal(policy)
	if err != nil {
		return "", err
	}

	return string(j), nil
}

func (s *server) filesystemUserCreatePolicy() (string, error) {
	policy := &iam.PolicyDocument{
		Version: "2012-10-17",
		Statement: []iam.StatementEntry{
			{
				Sid:    "CreateRepositoryUser",
				Effect: "Allow",
				Action: []string{
					"iam:CreatePolicy",
					"iam:UntagUser",
					"iam:GetPolicyVersion",
					"iam:AddUserToGroup",
					"iam:GetPolicy",
					"iam:ListAttachedGroupPolicies",
					"iam:ListGroupPolicies",
					"iam:AttachGroupPolicy",
					"iam:GetUser",
					"iam:CreatePolicyVersion",
					"iam:CreateUser",
					"iam:GetGroup",
					"iam:CreateGroup",
					"iam:TagUser",
				},
				Resource: []string{
					"arn:aws:iam::*:group/*",
					fmt.Sprintf("arn:aws:iam::*:policy/spinup/%s/*", s.org),
					fmt.Sprintf("arn:aws:iam::*:user/spinup/%s/*", s.org),
				},
			},
			{
				Sid:    "ListRepositoryUserPolicies",
				Effect: "Allow",
				Action: []string{
					"iam:ListPolicies",
				},
				Resource: []string{"*"},
			},
		},
	}

	j, err := json.Marshal(policy)
	if err != nil {
		return "", err
	}

	return string(j), nil
}

func (s *server) filesystemUserDeletePolicy() (string, error) {
	policy := &iam.PolicyDocument{
		Version: "2012-10-17",
		Statement: []iam.StatementEntry{
			{
				Sid:    "DeleteRepositoryUser",
				Effect: "Allow",
				Action: []string{
					"iam:DeleteAccessKey",
					"iam:RemoveUserFromGroup",
					"iam:ListAccessKeys",
					"iam:ListGroupsForUser",
					"iam:DeleteUser",
					"iam:GetUser",
				},
				Resource: []string{
					fmt.Sprintf("arn:aws:iam::*:user/spinup/%s/*", s.org),
				},
			},
		},
	}

	j, err := json.Marshal(policy)
	if err != nil {
		return "", err
	}

	return string(j), nil
}

func (s *server) filesystemUserUpdatePolicy() (string, error) {
	policy := &iam.PolicyDocument{
		Version: "2012-10-17",
		Statement: []iam.StatementEntry{
			{
				Sid:    "UpdateRepositoryUser",
				Effect: "Allow",
				Action: []string{
					"iam:UntagUser",
					"iam:DeleteAccessKey",
					"iam:RemoveUserFromGroup",
					"iam:TagUser",
					"iam:CreateAccessKey",
					"iam:ListAccessKeys",
				},
				Resource: []string{
					fmt.Sprintf("arn:aws:iam::*:user/spinup/%s/*", s.org),
				},
			},
		},
	}

	j, err := json.Marshal(policy)
	if err != nil {
		return "", err
	}

	return string(j), nil
}
