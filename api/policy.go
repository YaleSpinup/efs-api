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
					"iam:ListUsers",
					"iam:DeleteUser",
					"iam:GetUser",
				},
				Resource: []string{
					fmt.Sprintf("arn:aws:iam::*:user/spinup/%s/*", s.org),
					fmt.Sprintf("arn:aws:iam::*:group/spinup/%s/*", s.org),
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
					"iam:GetUser",
					"iam:UntagUser",
					"iam:DeleteAccessKey",
					"iam:RemoveUserFromGroup",
					"iam:TagUser",
					"iam:CreateAccessKey",
					"iam:ListAccessKeys",
				},
				Resource: []string{
					fmt.Sprintf("arn:aws:iam::*:user/spinup/%s/*", s.org),
					fmt.Sprintf("arn:aws:iam::*:group/spinup/%s/SpinupEFSAdminGroup-%s", s.org, s.org),
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

// efsPolicyFromFileSystemAccessPolicy constructs the EFS resource policy from the filesystem access policy flags
func efsPolicyFromFileSystemAccessPolicy(account, group, fsArn string, policy *FileSystemAccessPolicy) *iam.PolicyDocument {
	if policy == nil {
		return nil
	}

	policyDoc := iam.PolicyDocument{
		Version:   "2012-10-17",
		Id:        "efs-resource-policy-document",
		Statement: []iam.StatementEntry{},
	}

	if policy.EnforceEncryptedTransport {
		ep := iam.StatementEntry{
			Sid:       "DenyUnencryptedTransport",
			Effect:    "Deny",
			Principal: iam.Principal{"AWS": []string{"*"}},
			Action:    []string{"*"},
			Resource:  []string{fsArn},
			Condition: iam.Condition{
				"Bool": iam.ConditionStatement{
					"aws:SecureTransport": []string{"false"},
				},
			},
		}
		policyDoc.Statement = append(policyDoc.Statement, ep)
	}

	if policy.AllowAnonymousAccess {
		anonPolicy := iam.StatementEntry{
			Sid:       "AllowAnonymousAccess",
			Effect:    "Allow",
			Principal: iam.Principal{"AWS": []string{"*"}},
			Action: []string{
				"elasticfilesystem:ClientRootAccess",
				"elasticfilesystem:ClientWrite",
				"elasticfilesystem:ClientMount",
			},
			Resource: []string{fsArn},
		}
		policyDoc.Statement = append(policyDoc.Statement, anonPolicy)
	} else {
		anonPolicy := iam.StatementEntry{
			Sid:       "DenyAnonymousAccess",
			Effect:    "Allow",
			Principal: iam.Principal{"AWS": []string{"*"}},
			Action: []string{
				"elasticfilesystem:ClientRootAccess",
				"elasticfilesystem:ClientWrite",
			},
			Resource: []string{fsArn},
			Condition: iam.Condition{
				"Bool": iam.ConditionStatement{
					"elasticfilesystem:AccessedViaMountTarget": []string{"true"},
				},
			},
		}
		policyDoc.Statement = append(policyDoc.Statement, anonPolicy)
	}

	if policy.AllowEcsTaskExecutionRole {
		roleArn := fmt.Sprintf("arn:aws:iam::%s:role/%s-ecsTaskExecution", account, group)
		log.Debugf("generated ecs task execution role arn: %s", roleArn)

		ecsPolicy := iam.StatementEntry{
			Sid:    "AllowECSAccessFromHomeSpace",
			Effect: "Allow",
			Principal: iam.Principal{"AWS": []string{"*"}},
			Action: []string{
				"elasticfilesystem:ClientRootAccess",
				"elasticfilesystem:ClientWrite",
				"elasticfilesystem:ClientMount",
			},
			Resource: []string{fsArn},
			Condition: iam.Condition{
				"Bool": iam.ConditionStatement{
					"elasticfilesystem:AccessedViaMountTarget": []string{"true"},
				},
			},
		}
		policyDoc.Statement = append(policyDoc.Statement, ecsPolicy)
	}

	return &policyDoc
}

// filSystemAccessPolicyFromEfsPolicy naively constructs the filesystem access policy flags from the SIDs defined in the EFS policy doc
func filSystemAccessPolicyFromEfsPolicy(policy string) (*FileSystemAccessPolicy, error) {
	if policy == "" {
		return nil, nil
	}

	policyDoc := iam.PolicyDocument{}
	if err := json.Unmarshal([]byte(policy), &policyDoc); err != nil {
		return nil, err
	}

	accessPolicy := FileSystemAccessPolicy{
		AllowAnonymousAccess:      true,
		EnforceEncryptedTransport: false,
		AllowEcsTaskExecutionRole: false,
	}

	for _, s := range policyDoc.Statement {
		switch s.Sid {
		case "DenyUnencryptedTransport":
			accessPolicy.EnforceEncryptedTransport = true
		case "DenyAnonymousAccess":
			accessPolicy.AllowAnonymousAccess = false
		case "AllowECSAccessFromHomeSpace":
			accessPolicy.AllowEcsTaskExecutionRole = true
		}
	}

	return &accessPolicy, nil
}

func generatePolicy(actions ...string) (string, error) {
	log.Debugf("generating %v policy document", actions)

	policy := iam.PolicyDocument{
		Version: "2012-10-17",
		Statement: []iam.StatementEntry{
			{
				Effect:   "Allow",
				Action:   actions,
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