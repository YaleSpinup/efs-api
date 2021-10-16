package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/YaleSpinup/apierror"
	"github.com/YaleSpinup/aws-go/services/iam"
	"github.com/aws/aws-sdk-go/aws"
	log "github.com/sirupsen/logrus"
)

var efsAdminPolicyDoc string
var EfsAdminPolicy = iam.PolicyDocument{
	Version: "2012-10-17",
	Statement: []iam.StatementEntry{
		{
			Sid:    "AllowActionsOnVolumesInSpaceAndOrg",
			Effect: "Allow",
			Action: []string{
				"elasticfilesystem:ClientRootAccess",
				"elasticfilesystem:ClientWrite",
				"elasticfilesystem:ClientMount",
			},
			Resource: []string{"*"},
			Condition: iam.Condition{
				"StringEqualsIgnoreCase": iam.ConditionStatement{
					"aws:ResourceTag/Name":           []string{"${aws:PrincipalTag/ResourceName}"},
					"aws:ResourceTag/spinup:org":     []string{"${aws:PrincipalTag/spinup:org}"},
					"aws:ResourceTag/spinup:spaceid": []string{"${aws:PrincipalTag/spinup:spaceid}"},
				},
			},
		},
	},
}

// cachePolicyDoc generates the string value of the policy document the first time it's used and keeps
// it in memory to prevent marshalling static data on each request.
func cachePolicyDoc() error {
	// initialize ecr admin policy document
	policyDoc, err := json.Marshal(EfsAdminPolicy)
	if err != nil {
		return err
	}
	efsAdminPolicyDoc = string(policyDoc)
	return nil
}

// prepareAccount sets up the account for user management by creating the admin policy and group
func (o *userOrchestrator) prepareAccount(ctx context.Context) error {
	log.Info("preparing account for user management")

	path := fmt.Sprintf("/spinup/%s/", o.org)

	if efsAdminPolicyDoc == "" {
		if err := cachePolicyDoc(); err != nil {
			return err
		}
	}

	policyName := fmt.Sprintf("SpinupEFSAdminPolicy-%s", o.org)
	policyArn, err := o.userCreatePolicyIfMissing(ctx, policyName, path)
	if err != nil {
		return err
	}

	groupName := fmt.Sprintf("SpinupEFSAdminGroup-%s", o.org)
	if err := o.userCreateGroupIfMissing(ctx, groupName, path, policyArn); err != nil {
		return err
	}

	return err
}

// userCreatePolicyIfMissing gets the given policy by name.  if the policy isn't found it simply creates the policy and
// returns.  if the policy is found, it gets the policy document and compares to the expected policy document, updating
// if they differ.
func (o *userOrchestrator) userCreatePolicyIfMissing(ctx context.Context, name, path string) (string, error) {
	log.Infof("creating policy %s in %s if missing", name, path)

	policy, err := o.iamClient.GetPolicyByName(ctx, name, path)
	if err != nil {
		if aerr, ok := err.(apierror.Error); !ok || aerr.Code != apierror.ErrNotFound {
			return "", err
		}

		log.Infof("policy %s not found, creating", name)
	}

	// if the policy isn't found, create it and return
	if policy == nil {
		out, err := o.iamClient.CreatePolicy(ctx, name, path, efsAdminPolicyDoc)
		if err != nil {
			return "", err
		}

		if err := o.iamClient.WaitForPolicy(ctx, aws.StringValue(out.Arn)); err != nil {
			return "", err
		}

		return aws.StringValue(out.Arn), nil
	}

	out, err := o.iamClient.GetDefaultPolicyVersion(ctx, aws.StringValue(policy.Arn), aws.StringValue(policy.DefaultVersionId))
	if err != nil {
		return "", err
	}

	// Document is returned url encoded, we must decode it to unmarshal and compare
	d, err := url.QueryUnescape(aws.StringValue(out.Document))
	if err != nil {
		return "", err
	}

	// If we cannot unmarshal the document we received into an iam.PolicyDocument or if
	// the document doesn't match, let's try to update it.  If unmarshaling fails, we assume
	// our struct has changed (for example going from Resource string to Resource []string)
	var updatePolicy bool
	doc := iam.PolicyDocument{}
	if err := json.Unmarshal([]byte(d), &doc); err != nil {
		log.Warnf("error getting policy document: %s, updating", err)
		updatePolicy = true
	} else if !iam.PolicyDeepEqual(doc, EfsAdminPolicy) {
		log.Warn("policy document is not the same, updating")
		updatePolicy = true
	}

	if updatePolicy {
		if err := o.iamClient.UpdatePolicy(ctx, aws.StringValue(policy.Arn), efsAdminPolicyDoc); err != nil {
			return "", err
		}

		// TODO: delete old version, only 5 versions allowed
	}

	return aws.StringValue(policy.Arn), nil
}

// userCreateGroupIfMissing gets the group and creates it if it's missing.  it also checks if the correct
// policyArn is attached to the group, and attaches it if it's not.
func (o *userOrchestrator) userCreateGroupIfMissing(ctx context.Context, name, path, policyArn string) error {
	log.Infof("creating group %s in %s and assigning policy %s if missing", name, path, policyArn)

	if _, err := o.iamClient.GetGroupWithPath(ctx, name, path); err != nil {
		if aerr, ok := err.(apierror.Error); !ok || aerr.Code != apierror.ErrNotFound {
			return err
		}

		log.Infof("group %s not found, creating", name)

		if _, err := o.iamClient.CreateGroup(ctx, name, path); err != nil {
			return err
		}
	}

	// get the list of attached policies for the group
	attachedPolicies, err := o.iamClient.ListAttachedGroupPolicies(ctx, name, path)
	if err != nil {
		return err
	}

	// return if the policy is already attached to the group
	for _, p := range attachedPolicies {
		if p != policyArn {
			continue
		}

		return nil
	}

	if err := o.iamClient.AttachGroupPolicy(ctx, name, policyArn); err != nil {
		return err
	}

	return nil
}
