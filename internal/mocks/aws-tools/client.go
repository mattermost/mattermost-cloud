// Copyright (c) Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//
// Code generated by MockGen. DO NOT EDIT.
// Source: ../../tools/aws/client.go

// Package mockawstools is a generated GoMock package.
package mockawstools

import (
	types "github.com/aws/aws-sdk-go-v2/service/eks/types"
	gomock "github.com/golang/mock/gomock"
	aws "github.com/mattermost/mattermost-cloud/internal/tools/aws"
	model "github.com/mattermost/mattermost-cloud/model"
	logrus "github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	reflect "reflect"
)

// MockAWS is a mock of AWS interface
type MockAWS struct {
	ctrl     *gomock.Controller
	recorder *MockAWSMockRecorder
}

// MockAWSMockRecorder is the mock recorder for MockAWS
type MockAWSMockRecorder struct {
	mock *MockAWS
}

// NewMockAWS creates a new mock instance
func NewMockAWS(ctrl *gomock.Controller) *MockAWS {
	mock := &MockAWS{ctrl: ctrl}
	mock.recorder = &MockAWSMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockAWS) EXPECT() *MockAWSMockRecorder {
	return m.recorder
}

// GetCertificateSummaryByTag mocks base method
func (m *MockAWS) GetCertificateSummaryByTag(key, value string, logger logrus.FieldLogger) (*model.Certificate, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetCertificateSummaryByTag", key, value, logger)
	ret0, _ := ret[0].(*model.Certificate)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetCertificateSummaryByTag indicates an expected call of GetCertificateSummaryByTag
func (mr *MockAWSMockRecorder) GetCertificateSummaryByTag(key, value, logger interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetCertificateSummaryByTag", reflect.TypeOf((*MockAWS)(nil).GetCertificateSummaryByTag), key, value, logger)
}

// GetCloudEnvironmentName mocks base method
func (m *MockAWS) GetCloudEnvironmentName() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetCloudEnvironmentName")
	ret0, _ := ret[0].(string)
	return ret0
}

// GetCloudEnvironmentName indicates an expected call of GetCloudEnvironmentName
func (mr *MockAWSMockRecorder) GetCloudEnvironmentName() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetCloudEnvironmentName", reflect.TypeOf((*MockAWS)(nil).GetCloudEnvironmentName))
}

// GetAndClaimVpcResources mocks base method
func (m *MockAWS) GetAndClaimVpcResources(cluster *model.Cluster, owner string, logger logrus.FieldLogger) (aws.ClusterResources, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetAndClaimVpcResources", cluster, owner, logger)
	ret0, _ := ret[0].(aws.ClusterResources)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetAndClaimVpcResources indicates an expected call of GetAndClaimVpcResources
func (mr *MockAWSMockRecorder) GetAndClaimVpcResources(cluster, owner, logger interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetAndClaimVpcResources", reflect.TypeOf((*MockAWS)(nil).GetAndClaimVpcResources), cluster, owner, logger)
}

// ClaimVPC mocks base method
func (m *MockAWS) ClaimVPC(vpcID string, cluster *model.Cluster, owner string, logger logrus.FieldLogger) (aws.ClusterResources, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ClaimVPC", vpcID, cluster, owner, logger)
	ret0, _ := ret[0].(aws.ClusterResources)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ClaimVPC indicates an expected call of ClaimVPC
func (mr *MockAWSMockRecorder) ClaimVPC(vpcID, cluster, owner, logger interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ClaimVPC", reflect.TypeOf((*MockAWS)(nil).ClaimVPC), vpcID, cluster, owner, logger)
}

// ReleaseVpc mocks base method
func (m *MockAWS) ReleaseVpc(cluster *model.Cluster, logger logrus.FieldLogger) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ReleaseVpc", cluster, logger)
	ret0, _ := ret[0].(error)
	return ret0
}

// ReleaseVpc indicates an expected call of ReleaseVpc
func (mr *MockAWSMockRecorder) ReleaseVpc(cluster, logger interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ReleaseVpc", reflect.TypeOf((*MockAWS)(nil).ReleaseVpc), cluster, logger)
}

// GetClaimedVPC mocks base method
func (m *MockAWS) GetClaimedVPC(clusterID string, logger logrus.FieldLogger) (string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetClaimedVPC", clusterID, logger)
	ret0, _ := ret[0].(string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetClaimedVPC indicates an expected call of GetClaimedVPC
func (mr *MockAWSMockRecorder) GetClaimedVPC(clusterID, logger interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetClaimedVPC", reflect.TypeOf((*MockAWS)(nil).GetClaimedVPC), clusterID, logger)
}

// AttachPolicyToRole mocks base method
func (m *MockAWS) AttachPolicyToRole(roleName, policyName string, logger logrus.FieldLogger) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "AttachPolicyToRole", roleName, policyName, logger)
	ret0, _ := ret[0].(error)
	return ret0
}

// AttachPolicyToRole indicates an expected call of AttachPolicyToRole
func (mr *MockAWSMockRecorder) AttachPolicyToRole(roleName, policyName, logger interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "AttachPolicyToRole", reflect.TypeOf((*MockAWS)(nil).AttachPolicyToRole), roleName, policyName, logger)
}

// DetachPolicyFromRole mocks base method
func (m *MockAWS) DetachPolicyFromRole(roleName, policyName string, logger logrus.FieldLogger) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DetachPolicyFromRole", roleName, policyName, logger)
	ret0, _ := ret[0].(error)
	return ret0
}

// DetachPolicyFromRole indicates an expected call of DetachPolicyFromRole
func (mr *MockAWSMockRecorder) DetachPolicyFromRole(roleName, policyName, logger interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DetachPolicyFromRole", reflect.TypeOf((*MockAWS)(nil).DetachPolicyFromRole), roleName, policyName, logger)
}

// ClaimSecurityGroups mocks base method
func (m *MockAWS) ClaimSecurityGroups(cluster *model.Cluster, ngNames, vpcID string, logger logrus.FieldLogger) ([]string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ClaimSecurityGroups", cluster, ngNames, vpcID, logger)
	ret0, _ := ret[0].([]string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ClaimSecurityGroups indicates an expected call of ClaimSecurityGroups
func (mr *MockAWSMockRecorder) ClaimSecurityGroups(cluster, ngNames, vpcID, logger interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ClaimSecurityGroups", reflect.TypeOf((*MockAWS)(nil).ClaimSecurityGroups), cluster, ngNames, vpcID, logger)
}

// GetPrivateZoneDomainName mocks base method
func (m *MockAWS) GetPrivateZoneDomainName(logger logrus.FieldLogger) (string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetPrivateZoneDomainName", logger)
	ret0, _ := ret[0].(string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetPrivateZoneDomainName indicates an expected call of GetPrivateZoneDomainName
func (mr *MockAWSMockRecorder) GetPrivateZoneDomainName(logger interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetPrivateZoneDomainName", reflect.TypeOf((*MockAWS)(nil).GetPrivateZoneDomainName), logger)
}

// CreatePrivateCNAME mocks base method
func (m *MockAWS) CreatePrivateCNAME(dnsName string, dnsEndpoints []string, logger logrus.FieldLogger) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreatePrivateCNAME", dnsName, dnsEndpoints, logger)
	ret0, _ := ret[0].(error)
	return ret0
}

// CreatePrivateCNAME indicates an expected call of CreatePrivateCNAME
func (mr *MockAWSMockRecorder) CreatePrivateCNAME(dnsName, dnsEndpoints, logger interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreatePrivateCNAME", reflect.TypeOf((*MockAWS)(nil).CreatePrivateCNAME), dnsName, dnsEndpoints, logger)
}

// IsProvisionedPrivateCNAME mocks base method
func (m *MockAWS) IsProvisionedPrivateCNAME(dnsName string, logger logrus.FieldLogger) bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "IsProvisionedPrivateCNAME", dnsName, logger)
	ret0, _ := ret[0].(bool)
	return ret0
}

// IsProvisionedPrivateCNAME indicates an expected call of IsProvisionedPrivateCNAME
func (mr *MockAWSMockRecorder) IsProvisionedPrivateCNAME(dnsName, logger interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "IsProvisionedPrivateCNAME", reflect.TypeOf((*MockAWS)(nil).IsProvisionedPrivateCNAME), dnsName, logger)
}

// DeletePrivateCNAME mocks base method
func (m *MockAWS) DeletePrivateCNAME(dnsName string, logger logrus.FieldLogger) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeletePrivateCNAME", dnsName, logger)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeletePrivateCNAME indicates an expected call of DeletePrivateCNAME
func (mr *MockAWSMockRecorder) DeletePrivateCNAME(dnsName, logger interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeletePrivateCNAME", reflect.TypeOf((*MockAWS)(nil).DeletePrivateCNAME), dnsName, logger)
}

// DeletePublicCNAMEs mocks base method
func (m *MockAWS) DeletePublicCNAMEs(dnsName []string, logger logrus.FieldLogger) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeletePublicCNAMEs", dnsName, logger)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeletePublicCNAMEs indicates an expected call of DeletePublicCNAMEs
func (mr *MockAWSMockRecorder) DeletePublicCNAMEs(dnsName, logger interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeletePublicCNAMEs", reflect.TypeOf((*MockAWS)(nil).DeletePublicCNAMEs), dnsName, logger)
}

// UpsertPublicCNAMEs mocks base method
func (m *MockAWS) UpsertPublicCNAMEs(dnsNames, endpoints []string, logger logrus.FieldLogger) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpsertPublicCNAMEs", dnsNames, endpoints, logger)
	ret0, _ := ret[0].(error)
	return ret0
}

// UpsertPublicCNAMEs indicates an expected call of UpsertPublicCNAMEs
func (mr *MockAWSMockRecorder) UpsertPublicCNAMEs(dnsNames, endpoints, logger interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpsertPublicCNAMEs", reflect.TypeOf((*MockAWS)(nil).UpsertPublicCNAMEs), dnsNames, endpoints, logger)
}

// IsValidAMI mocks base method
func (m *MockAWS) IsValidAMI(AMIImage string, logger logrus.FieldLogger) (bool, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "IsValidAMI", AMIImage, logger)
	ret0, _ := ret[0].(bool)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// IsValidAMI indicates an expected call of IsValidAMI
func (mr *MockAWSMockRecorder) IsValidAMI(AMIImage, logger interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "IsValidAMI", reflect.TypeOf((*MockAWS)(nil).IsValidAMI), AMIImage, logger)
}

// GetAMIByTag mocks base method
func (m *MockAWS) GetAMIByTag(tagKey, tagValue string, logger logrus.FieldLogger) (string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetAMIByTag", tagKey, tagValue, logger)
	ret0, _ := ret[0].(string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetAMIByTag indicates an expected call of GetAMIByTag
func (mr *MockAWSMockRecorder) GetAMIByTag(tagKey, tagValue, logger interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetAMIByTag", reflect.TypeOf((*MockAWS)(nil).GetAMIByTag), tagKey, tagValue, logger)
}

// S3EnsureBucketDeleted mocks base method
func (m *MockAWS) S3EnsureBucketDeleted(bucketName string, logger logrus.FieldLogger) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "S3EnsureBucketDeleted", bucketName, logger)
	ret0, _ := ret[0].(error)
	return ret0
}

// S3EnsureBucketDeleted indicates an expected call of S3EnsureBucketDeleted
func (mr *MockAWSMockRecorder) S3EnsureBucketDeleted(bucketName, logger interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "S3EnsureBucketDeleted", reflect.TypeOf((*MockAWS)(nil).S3EnsureBucketDeleted), bucketName, logger)
}

// S3EnsureObjectDeleted mocks base method
func (m *MockAWS) S3EnsureObjectDeleted(bucketName, path string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "S3EnsureObjectDeleted", bucketName, path)
	ret0, _ := ret[0].(error)
	return ret0
}

// S3EnsureObjectDeleted indicates an expected call of S3EnsureObjectDeleted
func (mr *MockAWSMockRecorder) S3EnsureObjectDeleted(bucketName, path interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "S3EnsureObjectDeleted", reflect.TypeOf((*MockAWS)(nil).S3EnsureObjectDeleted), bucketName, path)
}

// S3LargeCopy mocks base method
func (m *MockAWS) S3LargeCopy(srcBucketName, srcKey, destBucketName, destKey *string, logger logrus.FieldLogger) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "S3LargeCopy", srcBucketName, srcKey, destBucketName, destKey, logger)
	ret0, _ := ret[0].(error)
	return ret0
}

// S3LargeCopy indicates an expected call of S3LargeCopy
func (mr *MockAWSMockRecorder) S3LargeCopy(srcBucketName, srcKey, destBucketName, destKey, logger interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "S3LargeCopy", reflect.TypeOf((*MockAWS)(nil).S3LargeCopy), srcBucketName, srcKey, destBucketName, destKey, logger)
}

// GetMultitenantBucketNameForInstallation mocks base method
func (m *MockAWS) GetMultitenantBucketNameForInstallation(installationID string, store model.InstallationDatabaseStoreInterface) (string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetMultitenantBucketNameForInstallation", installationID, store)
	ret0, _ := ret[0].(string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetMultitenantBucketNameForInstallation indicates an expected call of GetMultitenantBucketNameForInstallation
func (mr *MockAWSMockRecorder) GetMultitenantBucketNameForInstallation(installationID, store interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetMultitenantBucketNameForInstallation", reflect.TypeOf((*MockAWS)(nil).GetMultitenantBucketNameForInstallation), installationID, store)
}

// GetS3RegionURL mocks base method
func (m *MockAWS) GetS3RegionURL() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetS3RegionURL")
	ret0, _ := ret[0].(string)
	return ret0
}

// GetS3RegionURL indicates an expected call of GetS3RegionURL
func (mr *MockAWSMockRecorder) GetS3RegionURL() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetS3RegionURL", reflect.TypeOf((*MockAWS)(nil).GetS3RegionURL))
}

// GeneratePerseusUtilitySecret mocks base method
func (m *MockAWS) GeneratePerseusUtilitySecret(clusterID string, logger logrus.FieldLogger) (*v1.Secret, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GeneratePerseusUtilitySecret", clusterID, logger)
	ret0, _ := ret[0].(*v1.Secret)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GeneratePerseusUtilitySecret indicates an expected call of GeneratePerseusUtilitySecret
func (mr *MockAWSMockRecorder) GeneratePerseusUtilitySecret(clusterID, logger interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GeneratePerseusUtilitySecret", reflect.TypeOf((*MockAWS)(nil).GeneratePerseusUtilitySecret), clusterID, logger)
}

// GenerateBifrostUtilitySecret mocks base method
func (m *MockAWS) GenerateBifrostUtilitySecret(clusterID string, logger logrus.FieldLogger) (*v1.Secret, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GenerateBifrostUtilitySecret", clusterID, logger)
	ret0, _ := ret[0].(*v1.Secret)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GenerateBifrostUtilitySecret indicates an expected call of GenerateBifrostUtilitySecret
func (mr *MockAWSMockRecorder) GenerateBifrostUtilitySecret(clusterID, logger interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GenerateBifrostUtilitySecret", reflect.TypeOf((*MockAWS)(nil).GenerateBifrostUtilitySecret), clusterID, logger)
}

// GetCIDRByVPCTag mocks base method
func (m *MockAWS) GetCIDRByVPCTag(vpcTagName string, logger logrus.FieldLogger) (string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetCIDRByVPCTag", vpcTagName, logger)
	ret0, _ := ret[0].(string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetCIDRByVPCTag indicates an expected call of GetCIDRByVPCTag
func (mr *MockAWSMockRecorder) GetCIDRByVPCTag(vpcTagName, logger interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetCIDRByVPCTag", reflect.TypeOf((*MockAWS)(nil).GetCIDRByVPCTag), vpcTagName, logger)
}

// FixSubnetTagsForVPC mocks base method
func (m *MockAWS) FixSubnetTagsForVPC(vpc string, logger logrus.FieldLogger) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "FixSubnetTagsForVPC", vpc, logger)
	ret0, _ := ret[0].(error)
	return ret0
}

// FixSubnetTagsForVPC indicates an expected call of FixSubnetTagsForVPC
func (mr *MockAWSMockRecorder) FixSubnetTagsForVPC(vpc, logger interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "FixSubnetTagsForVPC", reflect.TypeOf((*MockAWS)(nil).FixSubnetTagsForVPC), vpc, logger)
}

// SecretsManagerGetPGBouncerAuthUserPassword mocks base method
func (m *MockAWS) SecretsManagerGetPGBouncerAuthUserPassword(vpcID string) (string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SecretsManagerGetPGBouncerAuthUserPassword", vpcID)
	ret0, _ := ret[0].(string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// SecretsManagerGetPGBouncerAuthUserPassword indicates an expected call of SecretsManagerGetPGBouncerAuthUserPassword
func (mr *MockAWSMockRecorder) SecretsManagerGetPGBouncerAuthUserPassword(vpcID interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SecretsManagerGetPGBouncerAuthUserPassword", reflect.TypeOf((*MockAWS)(nil).SecretsManagerGetPGBouncerAuthUserPassword), vpcID)
}

// SecretsManagerGetSecretBytes mocks base method
func (m *MockAWS) SecretsManagerGetSecretBytes(secretName string) ([]byte, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SecretsManagerGetSecretBytes", secretName)
	ret0, _ := ret[0].([]byte)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// SecretsManagerGetSecretBytes indicates an expected call of SecretsManagerGetSecretBytes
func (mr *MockAWSMockRecorder) SecretsManagerGetSecretBytes(secretName interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SecretsManagerGetSecretBytes", reflect.TypeOf((*MockAWS)(nil).SecretsManagerGetSecretBytes), secretName)
}

// SecretsManagerGetSecretAsK8sSecretData mocks base method
func (m *MockAWS) SecretsManagerGetSecretAsK8sSecretData(secretName string) (map[string][]byte, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SecretsManagerGetSecretAsK8sSecretData", secretName)
	ret0, _ := ret[0].(map[string][]byte)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// SecretsManagerGetSecretAsK8sSecretData indicates an expected call of SecretsManagerGetSecretAsK8sSecretData
func (mr *MockAWSMockRecorder) SecretsManagerGetSecretAsK8sSecretData(secretName interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SecretsManagerGetSecretAsK8sSecretData", reflect.TypeOf((*MockAWS)(nil).SecretsManagerGetSecretAsK8sSecretData), secretName)
}

// SecretsManagerEnsureSecretDeleted mocks base method
func (m *MockAWS) SecretsManagerEnsureSecretDeleted(secretName string, logger logrus.FieldLogger) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SecretsManagerEnsureSecretDeleted", secretName, logger)
	ret0, _ := ret[0].(error)
	return ret0
}

// SecretsManagerEnsureSecretDeleted indicates an expected call of SecretsManagerEnsureSecretDeleted
func (mr *MockAWSMockRecorder) SecretsManagerEnsureSecretDeleted(secretName, logger interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SecretsManagerEnsureSecretDeleted", reflect.TypeOf((*MockAWS)(nil).SecretsManagerEnsureSecretDeleted), secretName, logger)
}

// EnsureEKSCluster mocks base method
func (m *MockAWS) EnsureEKSCluster(cluster *model.Cluster, resources aws.ClusterResources) (*types.Cluster, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "EnsureEKSCluster", cluster, resources)
	ret0, _ := ret[0].(*types.Cluster)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// EnsureEKSCluster indicates an expected call of EnsureEKSCluster
func (mr *MockAWSMockRecorder) EnsureEKSCluster(cluster, resources interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "EnsureEKSCluster", reflect.TypeOf((*MockAWS)(nil).EnsureEKSCluster), cluster, resources)
}

// EnsureEKSClusterUpdated mocks base method
func (m *MockAWS) EnsureEKSClusterUpdated(cluster *model.Cluster) (*types.Update, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "EnsureEKSClusterUpdated", cluster)
	ret0, _ := ret[0].(*types.Update)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// EnsureEKSClusterUpdated indicates an expected call of EnsureEKSClusterUpdated
func (mr *MockAWSMockRecorder) EnsureEKSClusterUpdated(cluster interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "EnsureEKSClusterUpdated", reflect.TypeOf((*MockAWS)(nil).EnsureEKSClusterUpdated), cluster)
}

// EnsureEKSNodeGroup mocks base method
func (m *MockAWS) EnsureEKSNodeGroup(cluster *model.Cluster, nodeGroupPrefix string) (*types.Nodegroup, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "EnsureEKSNodeGroup", cluster, nodeGroupPrefix)
	ret0, _ := ret[0].(*types.Nodegroup)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// EnsureEKSNodeGroup indicates an expected call of EnsureEKSNodeGroup
func (mr *MockAWSMockRecorder) EnsureEKSNodeGroup(cluster, nodeGroupPrefix interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "EnsureEKSNodeGroup", reflect.TypeOf((*MockAWS)(nil).EnsureEKSNodeGroup), cluster, nodeGroupPrefix)
}

// EnsureEKSNodeGroupMigrated mocks base method
func (m *MockAWS) EnsureEKSNodeGroupMigrated(cluster *model.Cluster, nodeGroupPrefix string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "EnsureEKSNodeGroupMigrated", cluster, nodeGroupPrefix)
	ret0, _ := ret[0].(error)
	return ret0
}

// EnsureEKSNodeGroupMigrated indicates an expected call of EnsureEKSNodeGroupMigrated
func (mr *MockAWSMockRecorder) EnsureEKSNodeGroupMigrated(cluster, nodeGroupPrefix interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "EnsureEKSNodeGroupMigrated", reflect.TypeOf((*MockAWS)(nil).EnsureEKSNodeGroupMigrated), cluster, nodeGroupPrefix)
}

// GetActiveEKSCluster mocks base method
func (m *MockAWS) GetActiveEKSCluster(clusterName string) (*types.Cluster, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetActiveEKSCluster", clusterName)
	ret0, _ := ret[0].(*types.Cluster)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetActiveEKSCluster indicates an expected call of GetActiveEKSCluster
func (mr *MockAWSMockRecorder) GetActiveEKSCluster(clusterName interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetActiveEKSCluster", reflect.TypeOf((*MockAWS)(nil).GetActiveEKSCluster), clusterName)
}

// GetActiveEKSNodeGroup mocks base method
func (m *MockAWS) GetActiveEKSNodeGroup(clusterName, nodeGroupName string) (*types.Nodegroup, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetActiveEKSNodeGroup", clusterName, nodeGroupName)
	ret0, _ := ret[0].(*types.Nodegroup)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetActiveEKSNodeGroup indicates an expected call of GetActiveEKSNodeGroup
func (mr *MockAWSMockRecorder) GetActiveEKSNodeGroup(clusterName, nodeGroupName interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetActiveEKSNodeGroup", reflect.TypeOf((*MockAWS)(nil).GetActiveEKSNodeGroup), clusterName, nodeGroupName)
}

// EnsureEKSNodeGroupDeleted mocks base method
func (m *MockAWS) EnsureEKSNodeGroupDeleted(clusterName, nodeGroupName string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "EnsureEKSNodeGroupDeleted", clusterName, nodeGroupName)
	ret0, _ := ret[0].(error)
	return ret0
}

// EnsureEKSNodeGroupDeleted indicates an expected call of EnsureEKSNodeGroupDeleted
func (mr *MockAWSMockRecorder) EnsureEKSNodeGroupDeleted(clusterName, nodeGroupName interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "EnsureEKSNodeGroupDeleted", reflect.TypeOf((*MockAWS)(nil).EnsureEKSNodeGroupDeleted), clusterName, nodeGroupName)
}

// EnsureEKSClusterDeleted mocks base method
func (m *MockAWS) EnsureEKSClusterDeleted(clusterName string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "EnsureEKSClusterDeleted", clusterName)
	ret0, _ := ret[0].(error)
	return ret0
}

// EnsureEKSClusterDeleted indicates an expected call of EnsureEKSClusterDeleted
func (mr *MockAWSMockRecorder) EnsureEKSClusterDeleted(clusterName interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "EnsureEKSClusterDeleted", reflect.TypeOf((*MockAWS)(nil).EnsureEKSClusterDeleted), clusterName)
}

// InstallEKSAddons mocks base method
func (m *MockAWS) InstallEKSAddons(cluster *model.Cluster) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "InstallEKSAddons", cluster)
	ret0, _ := ret[0].(error)
	return ret0
}

// InstallEKSAddons indicates an expected call of InstallEKSAddons
func (mr *MockAWSMockRecorder) InstallEKSAddons(cluster interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "InstallEKSAddons", reflect.TypeOf((*MockAWS)(nil).InstallEKSAddons), cluster)
}

// WaitForActiveEKSCluster mocks base method
func (m *MockAWS) WaitForActiveEKSCluster(clusterName string, timeout int) (*types.Cluster, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "WaitForActiveEKSCluster", clusterName, timeout)
	ret0, _ := ret[0].(*types.Cluster)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// WaitForActiveEKSCluster indicates an expected call of WaitForActiveEKSCluster
func (mr *MockAWSMockRecorder) WaitForActiveEKSCluster(clusterName, timeout interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "WaitForActiveEKSCluster", reflect.TypeOf((*MockAWS)(nil).WaitForActiveEKSCluster), clusterName, timeout)
}

// WaitForActiveEKSNodeGroup mocks base method
func (m *MockAWS) WaitForActiveEKSNodeGroup(clusterName, nodeGroupName string, timeout int) (*types.Nodegroup, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "WaitForActiveEKSNodeGroup", clusterName, nodeGroupName, timeout)
	ret0, _ := ret[0].(*types.Nodegroup)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// WaitForActiveEKSNodeGroup indicates an expected call of WaitForActiveEKSNodeGroup
func (mr *MockAWSMockRecorder) WaitForActiveEKSNodeGroup(clusterName, nodeGroupName, timeout interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "WaitForActiveEKSNodeGroup", reflect.TypeOf((*MockAWS)(nil).WaitForActiveEKSNodeGroup), clusterName, nodeGroupName, timeout)
}

// WaitForEKSNodeGroupToBeDeleted mocks base method
func (m *MockAWS) WaitForEKSNodeGroupToBeDeleted(clusterName, nodeGroupName string, timeout int) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "WaitForEKSNodeGroupToBeDeleted", clusterName, nodeGroupName, timeout)
	ret0, _ := ret[0].(error)
	return ret0
}

// WaitForEKSNodeGroupToBeDeleted indicates an expected call of WaitForEKSNodeGroupToBeDeleted
func (mr *MockAWSMockRecorder) WaitForEKSNodeGroupToBeDeleted(clusterName, nodeGroupName, timeout interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "WaitForEKSNodeGroupToBeDeleted", reflect.TypeOf((*MockAWS)(nil).WaitForEKSNodeGroupToBeDeleted), clusterName, nodeGroupName, timeout)
}

// WaitForEKSClusterToBeDeleted mocks base method
func (m *MockAWS) WaitForEKSClusterToBeDeleted(clusterName string, timeout int) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "WaitForEKSClusterToBeDeleted", clusterName, timeout)
	ret0, _ := ret[0].(error)
	return ret0
}

// WaitForEKSClusterToBeDeleted indicates an expected call of WaitForEKSClusterToBeDeleted
func (mr *MockAWSMockRecorder) WaitForEKSClusterToBeDeleted(clusterName, timeout interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "WaitForEKSClusterToBeDeleted", reflect.TypeOf((*MockAWS)(nil).WaitForEKSClusterToBeDeleted), clusterName, timeout)
}

// WaitForEKSClusterUpdateToBeCompleted mocks base method
func (m *MockAWS) WaitForEKSClusterUpdateToBeCompleted(clusterName, updateID string, timeout int) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "WaitForEKSClusterUpdateToBeCompleted", clusterName, updateID, timeout)
	ret0, _ := ret[0].(error)
	return ret0
}

// WaitForEKSClusterUpdateToBeCompleted indicates an expected call of WaitForEKSClusterUpdateToBeCompleted
func (mr *MockAWSMockRecorder) WaitForEKSClusterUpdateToBeCompleted(clusterName, updateID, timeout interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "WaitForEKSClusterUpdateToBeCompleted", reflect.TypeOf((*MockAWS)(nil).WaitForEKSClusterUpdateToBeCompleted), clusterName, updateID, timeout)
}

// CreateLaunchTemplate mocks base method
func (m *MockAWS) CreateLaunchTemplate(data *model.LaunchTemplateData) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateLaunchTemplate", data)
	ret0, _ := ret[0].(error)
	return ret0
}

// CreateLaunchTemplate indicates an expected call of CreateLaunchTemplate
func (mr *MockAWSMockRecorder) CreateLaunchTemplate(data interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateLaunchTemplate", reflect.TypeOf((*MockAWS)(nil).CreateLaunchTemplate), data)
}

// IsLaunchTemplateAvailable mocks base method
func (m *MockAWS) IsLaunchTemplateAvailable(launchTemplateName string) (bool, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "IsLaunchTemplateAvailable", launchTemplateName)
	ret0, _ := ret[0].(bool)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// IsLaunchTemplateAvailable indicates an expected call of IsLaunchTemplateAvailable
func (mr *MockAWSMockRecorder) IsLaunchTemplateAvailable(launchTemplateName interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "IsLaunchTemplateAvailable", reflect.TypeOf((*MockAWS)(nil).IsLaunchTemplateAvailable), launchTemplateName)
}

// UpdateLaunchTemplate mocks base method
func (m *MockAWS) UpdateLaunchTemplate(data *model.LaunchTemplateData) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateLaunchTemplate", data)
	ret0, _ := ret[0].(error)
	return ret0
}

// UpdateLaunchTemplate indicates an expected call of UpdateLaunchTemplate
func (mr *MockAWSMockRecorder) UpdateLaunchTemplate(data interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateLaunchTemplate", reflect.TypeOf((*MockAWS)(nil).UpdateLaunchTemplate), data)
}

// DeleteLaunchTemplate mocks base method
func (m *MockAWS) DeleteLaunchTemplate(launchTemplateName string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteLaunchTemplate", launchTemplateName)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteLaunchTemplate indicates an expected call of DeleteLaunchTemplate
func (mr *MockAWSMockRecorder) DeleteLaunchTemplate(launchTemplateName interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteLaunchTemplate", reflect.TypeOf((*MockAWS)(nil).DeleteLaunchTemplate), launchTemplateName)
}

// GetRegion mocks base method
func (m *MockAWS) GetRegion() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetRegion")
	ret0, _ := ret[0].(string)
	return ret0
}

// GetRegion indicates an expected call of GetRegion
func (mr *MockAWSMockRecorder) GetRegion() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetRegion", reflect.TypeOf((*MockAWS)(nil).GetRegion))
}

// GetAccountID mocks base method
func (m *MockAWS) GetAccountID() (string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetAccountID")
	ret0, _ := ret[0].(string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetAccountID indicates an expected call of GetAccountID
func (mr *MockAWSMockRecorder) GetAccountID() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetAccountID", reflect.TypeOf((*MockAWS)(nil).GetAccountID))
}

// GetLoadBalancerAPIByType mocks base method
func (m *MockAWS) GetLoadBalancerAPIByType(arg0 string) aws.ELB {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetLoadBalancerAPIByType", arg0)
	ret0, _ := ret[0].(aws.ELB)
	return ret0
}

// GetLoadBalancerAPIByType indicates an expected call of GetLoadBalancerAPIByType
func (mr *MockAWSMockRecorder) GetLoadBalancerAPIByType(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetLoadBalancerAPIByType", reflect.TypeOf((*MockAWS)(nil).GetLoadBalancerAPIByType), arg0)
}
