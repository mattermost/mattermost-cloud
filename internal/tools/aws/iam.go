// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package aws

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/iam/types"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

type policyDocument struct {
	Version   string
	Statement []policyStatementEntry
}

type policyStatementEntry struct {
	Sid       string
	Effect    string
	Action    []string
	Resource  string
	Condition policyStatementCondition `json:"Condition,omitempty"`
}

type policyStatementCondition map[string]map[string][]string

func (a *Client) iamEnsureUserCreated(awsID string, logger log.FieldLogger) (*types.User, error) {
	ctx := context.TODO()
	getResult, err := a.Service().iam.GetUser(
		ctx,
		&iam.GetUserInput{
			UserName: aws.String(awsID),
		})
	if err == nil {
		logger.WithField("iam-user-name", *getResult.User.UserName).Debug("AWS IAM user already created")
		return getResult.User, nil
	}
	var awsErr *types.NoSuchEntityException
	if errors.As(err, &awsErr) {
		return nil, err
	} else {
		return nil, err
	}

	createResult, err := a.Service().iam.CreateUser(
		ctx,
		&iam.CreateUserInput{
			UserName: aws.String(awsID),
		})
	if err != nil {
		return nil, err
	}

	logger.WithField("iam-user-name", *createResult.User.UserName).Debug("AWS IAM user created")

	return createResult.User, nil
}

func (a *Client) iamEnsureUserDeleted(awsID string, logger log.FieldLogger) error {
	ctx := context.TODO()
	_, err := a.Service().iam.GetUser(
		ctx,
		&iam.GetUserInput{
			UserName: aws.String(awsID),
		})
	if err != nil {
		var awsErr *types.NoSuchEntityException
		if errors.As(err, &awsErr) {
			logger.WithField("iam-user-name", awsID).Warn("AWS IAM user could not be found; assuming already deleted")
			return nil
		}

		return err
	}

	policyResult, err := a.Service().iam.ListAttachedUserPolicies(
		ctx,
		&iam.ListAttachedUserPoliciesInput{
			UserName: aws.String(awsID),
		})
	if err != nil {
		return err
	}
	for _, policy := range policyResult.AttachedPolicies {
		_, err = a.Service().iam.DetachUserPolicy(
			ctx,
			&iam.DetachUserPolicyInput{
				PolicyArn: policy.PolicyArn,
				UserName:  aws.String(awsID),
			})
		if err != nil {
			return err
		}

		logger.WithFields(log.Fields{
			"iam-user-name":   awsID,
			"iam-policy-name": *policy.PolicyName,
		}).Debug("AWS IAM policy detached from user")

		_, err = a.Service().iam.DeletePolicy(
			ctx,
			&iam.DeletePolicyInput{
				PolicyArn: policy.PolicyArn,
			})
		if err != nil {
			return err
		}

		logger.WithFields(log.Fields{
			"iam-user-name":   awsID,
			"iam-policy-name": *policy.PolicyName,
		}).Debug("AWS IAM policy deleted")
	}

	accessKeyResult, err := a.Service().iam.ListAccessKeys(
		ctx,
		&iam.ListAccessKeysInput{
			UserName: aws.String(awsID),
		})
	if err != nil {
		return err
	}
	for _, ak := range accessKeyResult.AccessKeyMetadata {
		_, err = a.Service().iam.DeleteAccessKey(
			ctx,
			&iam.DeleteAccessKeyInput{
				AccessKeyId: ak.AccessKeyId,
				UserName:    aws.String(awsID),
			})
		if err != nil {
			return err
		}

		logger.WithFields(log.Fields{
			"iam-user-name":     awsID,
			"iam-access-key-id": *ak.AccessKeyId,
		}).Debug("AWS IAM user access key deleted")
	}

	_, err = a.Service().iam.DeleteUser(
		ctx,
		&iam.DeleteUserInput{
			UserName: aws.String(awsID),
		})
	if err != nil {
		return err
	}

	logger.WithField("iam-user-name", awsID).Debug("AWS IAM user deleted")

	return nil
}

func (a *Client) iamEnsureS3PolicyCreated(awsID, policyARN, bucketName, permittedDirectory string, logger log.FieldLogger) (*types.Policy, error) {
	ctx := context.TODO()
	getResult, err := a.Service().iam.GetPolicy(
		ctx,
		&iam.GetPolicyInput{
			PolicyArn: aws.String(policyARN),
		})
	if err == nil {
		logger.WithField("iam-policy-name", *getResult.Policy.PolicyName).Debug("AWS IAM policy already created")
		return getResult.Policy, nil
	}
	if err != nil {
		var awsErr *types.NoSuchEntityException
		if errors.As(err, &awsErr) {
			return nil, err
		}
	} else {
		return nil, err
	}

	// The list condition directory needs a bit of logic to set correctly for
	// the single and multi-tenant S3 filestores.
	listCondition := policyStatementCondition{}
	if permittedDirectory != "*" {
		permittedDirectory = fmt.Sprintf("%s/*", permittedDirectory)
		listCondition = policyStatementCondition{
			"StringLike": {
				"s3:prefix": []string{permittedDirectory},
			},
		}
	}

	policy := policyDocument{
		Version: "2012-10-17",
		Statement: []policyStatementEntry{
			{
				Sid:    "ListObjectsInBucket",
				Effect: "Allow",
				Action: []string{
					"s3:ListBucket",
				},
				Resource:  fmt.Sprintf("arn:aws:s3:::%s", bucketName),
				Condition: listCondition,
			}, {
				Sid:    "AllObjectActions",
				Effect: "Allow",
				Action: []string{
					"s3:GetObject",
					"s3:PutObject",
					"s3:ListBucket",
					"s3:PutObjectAcl",
					"s3:DeleteObject",
				},
				Resource: fmt.Sprintf("arn:aws:s3:::%s/%s", bucketName, permittedDirectory),
			},
		},
	}

	b, err := json.Marshal(&policy)
	if err != nil {
		return nil, errors.Wrap(err, "unable to marshal IAM policy")
	}

	createResult, err := a.Service().iam.CreatePolicy(
		ctx,
		&iam.CreatePolicyInput{
			PolicyDocument: aws.String(string(b)),
			PolicyName:     aws.String(awsID),
		})
	if err != nil {
		return nil, errors.Wrap(err, "unable to create IAM policy")
	}

	logger.WithField("iam-policy-name", *createResult.Policy.PolicyName).Debug("AWS IAM policy created")

	return createResult.Policy, nil
}

func (a *Client) iamEnsurePolicyAttached(awsID, policyARN string, logger log.FieldLogger) error {
	_, err := a.Service().iam.AttachUserPolicy(
		context.TODO(),
		&iam.AttachUserPolicyInput{
			PolicyArn: aws.String(policyARN),
			UserName:  aws.String(awsID),
		})
	if err != nil {
		return err
	}

	return nil
}

func (a *Client) iamEnsureAccessKeyCreated(awsID string, logger log.FieldLogger) (*types.AccessKey, error) {
	ctx := context.TODO()
	listResult, err := a.Service().iam.ListAccessKeys(
		ctx,
		&iam.ListAccessKeysInput{
			UserName: aws.String(awsID),
		})
	if err != nil {
		return nil, err
	}
	for _, ak := range listResult.AccessKeyMetadata {
		_, err = a.Service().iam.DeleteAccessKey(
			ctx,
			&iam.DeleteAccessKeyInput{
				AccessKeyId: ak.AccessKeyId,
				UserName:    aws.String(awsID),
			})
		if err != nil {
			return nil, err
		}

		logger.WithFields(log.Fields{
			"iam-user-name":     awsID,
			"iam-access-key-id": *ak.AccessKeyId,
		}).Info("AWS IAM user access key deleted")
	}

	createResult, err := a.Service().iam.CreateAccessKey(
		ctx,
		&iam.CreateAccessKeyInput{
			UserName: aws.String(awsID),
		})
	if err != nil {
		return nil, err
	}

	return createResult.AccessKey, nil
}

// GetAccountAliases returns the AWS account name aliases.
func (a *Client) GetAccountAliases() (*iam.ListAccountAliasesOutput, error) {
	accountAliases, err := a.Service().iam.ListAccountAliases(context.TODO(), &iam.ListAccountAliasesInput{})
	if err != nil {
		return nil, errors.Wrap(err, "unable to get AWS account name aliases")
	}
	return accountAliases, nil
}

// AttachPolicyToRole attaches a pre-created IAM policy to an IAM role.
func (a *Client) AttachPolicyToRole(roleName, policyName string, logger log.FieldLogger) error {
	accountID, err := a.GetAccountID()
	if err != nil {
		return errors.Wrap(err, "unable to get the current AWS Account ID")
	}
	policyARN := fmt.Sprintf("arn:aws:iam::%s:policy/%s", accountID, policyName)

	logger.Infof("Attaching policy (%s) to IAM role (%s)", policyARN, roleName)
	_, err = a.Service().iam.AttachRolePolicy(
		context.TODO(),
		&iam.AttachRolePolicyInput{
			PolicyArn: aws.String(policyARN),
			RoleName:  aws.String(roleName),
		})
	if err != nil {
		return errors.Wrap(err, "unable to attach policy to IAM role")
	}
	logger.Info("IAM Policy successfully attached")
	return nil
}

// DetachPolicyFromRole detaches an IAM policy from an IAM role.
func (a *Client) DetachPolicyFromRole(roleName, policyName string, logger log.FieldLogger) error {
	accountID, err := a.GetAccountID()
	if err != nil {
		return errors.Wrap(err, "unable to get the current AWS Account ID")
	}
	policyARN := fmt.Sprintf("arn:aws:iam::%s:policy/%s", accountID, policyName)

	logger.Infof("Dettaching policy (%s) from IAM role (%s)", policyARN, roleName)
	_, err = a.Service().iam.DetachRolePolicy(
		context.TODO(),
		&iam.DetachRolePolicyInput{
			PolicyArn: aws.String(policyARN),
			RoleName:  aws.String(roleName),
		})
	var awsErr *types.NoSuchEntityException
	if errors.As(err, &awsErr) {
		logger.WithField("iam-policy", policyARN).Warn("IAM policy could not be detached; assuming already detached")
		return nil
	}
	if err != nil {
		return errors.Wrap(err, "unable to detach policy to IAM role")
	}
	logger.Info("IAM Policy successfully detached")
	return nil
}
