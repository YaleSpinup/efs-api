package api

import (
	"github.com/YaleSpinup/aws-go/services/iam"
)

type iamOrchestrator struct {
	client iam.IAM
	org    string
}

func newIamOrchestrator(client iam.IAM, org string) *iamOrchestrator {
	return &iamOrchestrator{
		client: client,
		org:    org,
	}
}
