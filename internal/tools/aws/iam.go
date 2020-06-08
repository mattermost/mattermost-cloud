package aws

import (
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

type policyDocument struct {
	Version   string
	Statement []policyStatementEntry
}

type policyStatementEntry struct {
	Sid      string
	Effect   string
	Action   []string
	Resource string
}

func (a *Client) iamEnsureUserCreated(awsID string, logger log.FieldLogger) (*iam.User, error) {
	getResult, err := a.Service().iam.GetUser(&iam.GetUserInput{
		UserName: aws.String(awsID),
	})
	if err == nil {
		logger.WithField("iam-user-name", *getResult.User.UserName).Debug("AWS IAM user already created")
		return getResult.User, nil
	}
	if aerr, ok := err.(awserr.Error); ok {
		if aerr.Code() != iam.ErrCodeNoSuchEntityException {
			return nil, err
		}
	} else {
		return nil, err
	}

	createResult, err := a.Service().iam.CreateUser(&iam.CreateUserInput{
		UserName: aws.String(awsID),
	})
	if err != nil {
		return nil, err
	}

	logger.WithField("iam-user-name", *createResult.User.UserName).Debug("AWS IAM user created")

	return createResult.User, nil
}

func (a *Client) iamEnsureUserDeleted(awsID string, logger log.FieldLogger) error {
	_, err := a.Service().iam.GetUser(&iam.GetUserInput{
		UserName: aws.String(awsID),
	})
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			if aerr.Code() != iam.ErrCodeNoSuchEntityException {
				return err
			}

			logger.WithField("iam-user-name", awsID).Warn("AWS IAM user could not be found; assuming already deleted")
			return nil
		}
		return err
	}

	policyResult, err := a.Service().iam.ListAttachedUserPolicies(&iam.ListAttachedUserPoliciesInput{
		UserName: aws.String(awsID),
	})
	if err != nil {
		return err
	}
	for _, policy := range policyResult.AttachedPolicies {
		_, err = a.Service().iam.DetachUserPolicy(&iam.DetachUserPolicyInput{
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

		_, err = a.Service().iam.DeletePolicy(&iam.DeletePolicyInput{
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

	accessKeyResult, err := a.Service().iam.ListAccessKeys(&iam.ListAccessKeysInput{
		UserName: aws.String(awsID),
	})
	if err != nil {
		return err
	}
	for _, ak := range accessKeyResult.AccessKeyMetadata {
		_, err = a.Service().iam.DeleteAccessKey(&iam.DeleteAccessKeyInput{
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

	_, err = a.Service().iam.DeleteUser(&iam.DeleteUserInput{
		UserName: aws.String(awsID),
	})
	if err != nil {
		return err
	}

	logger.WithField("iam-user-name", awsID).Debug("AWS IAM user deleted")

	return nil
}

func (a *Client) iamEnsurePolicyCreated(awsID, policyARN string, logger log.FieldLogger) (*iam.Policy, error) {
	getResult, err := a.Service().iam.GetPolicy(&iam.GetPolicyInput{
		PolicyArn: aws.String(policyARN),
	})
	if err == nil {
		logger.WithField("iam-policy-name", *getResult.Policy.PolicyName).Debug("AWS IAM policy already created")
		return getResult.Policy, nil
	}
	if aerr, ok := err.(awserr.Error); ok {
		if aerr.Code() != iam.ErrCodeNoSuchEntityException {
			return nil, err
		}
	} else {
		return nil, err
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
				Resource: fmt.Sprintf("arn:aws:s3:::%s", awsID),
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
				Resource: fmt.Sprintf("arn:aws:s3:::%s/*", awsID),
			},
		},
	}

	b, err := json.Marshal(&policy)
	if err != nil {
		return nil, errors.Wrap(err, "unable to marshal IAM policy")
	}

	createResult, err := a.Service().iam.CreatePolicy(&iam.CreatePolicyInput{
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
	_, err := a.Service().iam.AttachUserPolicy(&iam.AttachUserPolicyInput{
		PolicyArn: aws.String(policyARN),
		UserName:  aws.String(awsID),
	})
	if err != nil {
		return err
	}

	return nil
}

func (a *Client) iamEnsureAccessKeyCreated(awsID string, logger log.FieldLogger) (*iam.AccessKey, error) {
	listResult, err := a.Service().iam.ListAccessKeys(&iam.ListAccessKeysInput{
		UserName: aws.String(awsID),
	})
	if err != nil {
		return nil, err
	}
	for _, ak := range listResult.AccessKeyMetadata {
		_, err = a.Service().iam.DeleteAccessKey(&iam.DeleteAccessKeyInput{
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

	createResult, err := a.Service().iam.CreateAccessKey(&iam.CreateAccessKeyInput{
		UserName: aws.String(awsID),
	})
	if err != nil {
		return nil, err
	}

	return createResult.AccessKey, nil
}

// GetAccountAliases returns the AWS account name aliases.
func (a *Client) GetAccountAliases() (*iam.ListAccountAliasesOutput, error) {
	accountAliases, err := a.Service().iam.ListAccountAliases(&iam.ListAccountAliasesInput{})
	if err != nil {
		return nil, errors.Wrap(err, "unable to get AWS account name aliases")
	}
	return accountAliases, nil
}
