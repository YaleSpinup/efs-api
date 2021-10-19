package api

import (
	"github.com/YaleSpinup/aws-go/services/iam"
	"github.com/YaleSpinup/efs-api/efs"
)

type userOrchestrator struct {
	iamClient iam.IAM
	efsClient efs.EFS
	org       string
}

func newUserOrchestrator(iamClient iam.IAM, efsClient efs.EFS, org string) *userOrchestrator {
	return &userOrchestrator{
		iamClient: iamClient,
		efsClient: efsClient,
		org:       org,
	}
}
