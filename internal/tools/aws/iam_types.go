// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package aws

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/iam"
)

// IAMAPI represents the series of calls we require from the AWS SDK v2 IAM Client
type IAMAPI interface {
	CreateUser(ctx context.Context, params *iam.CreateUserInput, optFns ...func(*iam.Options)) (*iam.CreateUserOutput, error)
	DeleteUser(ctx context.Context, params *iam.DeleteUserInput, optFns ...func(*iam.Options)) (*iam.DeleteUserOutput, error)
	GetUser(ctx context.Context, params *iam.GetUserInput, optFns ...func(*iam.Options)) (*iam.GetUserOutput, error)

	AttachUserPolicy(ctx context.Context, params *iam.AttachUserPolicyInput, optFns ...func(*iam.Options)) (*iam.AttachUserPolicyOutput, error)
	ListAttachedUserPolicies(ctx context.Context, params *iam.ListAttachedUserPoliciesInput, optFns ...func(*iam.Options)) (*iam.ListAttachedUserPoliciesOutput, error)
	DetachUserPolicy(ctx context.Context, params *iam.DetachUserPolicyInput, optFns ...func(*iam.Options)) (*iam.DetachUserPolicyOutput, error)

	ListAccountAliases(ctx context.Context, params *iam.ListAccountAliasesInput, optFns ...func(*iam.Options)) (*iam.ListAccountAliasesOutput, error)

	CreateAccessKey(ctx context.Context, params *iam.CreateAccessKeyInput, optFns ...func(*iam.Options)) (*iam.CreateAccessKeyOutput, error)
	DeleteAccessKey(ctx context.Context, params *iam.DeleteAccessKeyInput, optFns ...func(*iam.Options)) (*iam.DeleteAccessKeyOutput, error)
	ListAccessKeys(ctx context.Context, params *iam.ListAccessKeysInput, optFns ...func(*iam.Options)) (*iam.ListAccessKeysOutput, error)

	AttachRolePolicy(ctx context.Context, params *iam.AttachRolePolicyInput, optFns ...func(*iam.Options)) (*iam.AttachRolePolicyOutput, error)
	DetachRolePolicy(ctx context.Context, params *iam.DetachRolePolicyInput, optFns ...func(*iam.Options)) (*iam.DetachRolePolicyOutput, error)

	CreatePolicy(ctx context.Context, params *iam.CreatePolicyInput, optFns ...func(*iam.Options)) (*iam.CreatePolicyOutput, error)
	DeletePolicy(ctx context.Context, params *iam.DeletePolicyInput, optFns ...func(*iam.Options)) (*iam.DeletePolicyOutput, error)
	GetPolicy(ctx context.Context, params *iam.GetPolicyInput, optFns ...func(*iam.Options)) (*iam.GetPolicyOutput, error)
}
