// Copyright (c) Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//
// Code generated by MockGen. DO NOT EDIT.
// Source: ../../tools/aws/client.go

// Package mockawstools is a generated GoMock package.
package mockawstools

import (
	types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	types0 "github.com/aws/aws-sdk-go-v2/service/eks/types"
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
func (m *MockAWS) GetAndClaimVpcResources(cluster *model.Cluster, owner string, logger logrus.FieldLogger) (model.ClusterResources, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetAndClaimVpcResources", cluster, owner, logger)
	ret0, _ := ret[0].(model.ClusterResources)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetAndClaimVpcResources indicates an expected call of GetAndClaimVpcResources
func (mr *MockAWSMockRecorder) GetAndClaimVpcResources(cluster, owner, logger interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetAndClaimVpcResources", reflect.TypeOf((*MockAWS)(nil).GetAndClaimVpcResources), cluster, owner, logger)
}

// ClaimVPC mocks base method
func (m *MockAWS) ClaimVPC(vpcID string, cluster *model.Cluster, owner string, logger logrus.FieldLogger) (model.ClusterResources, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ClaimVPC", vpcID, cluster, owner, logger)
	ret0, _ := ret[0].(model.ClusterResources)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ClaimVPC indicates an expected call of ClaimVPC
func (mr *MockAWSMockRecorder) ClaimVPC(vpcID, cluster, owner, logger interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ClaimVPC", reflect.TypeOf((*MockAWS)(nil).ClaimVPC), vpcID, cluster, owner, logger)
}

// GetVpcResources mocks base method
func (m *MockAWS) GetVpcResources(clusterID string, logger logrus.FieldLogger) (model.ClusterResources, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetVpcResources", clusterID, logger)
	ret0, _ := ret[0].(model.ClusterResources)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetVpcResources indicates an expected call of GetVpcResources
func (mr *MockAWSMockRecorder) GetVpcResources(clusterID, logger interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetVpcResources", reflect.TypeOf((*MockAWS)(nil).GetVpcResources), clusterID, logger)
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

// FilterClusterResources mocks base method
func (m *MockAWS) FilterClusterResources(cluster *model.Cluster, resources model.ClusterResources) (model.ClusterResources, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "FilterClusterResources", cluster, resources)
	ret0, _ := ret[0].(model.ClusterResources)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// FilterClusterResources indicates an expected call of FilterClusterResources
func (mr *MockAWSMockRecorder) FilterClusterResources(cluster, resources interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "FilterClusterResources", reflect.TypeOf((*MockAWS)(nil).FilterClusterResources), cluster, resources)
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

// GetPrivateHostedZoneID mocks base method
func (m *MockAWS) GetPrivateHostedZoneID() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetPrivateHostedZoneID")
	ret0, _ := ret[0].(string)
	return ret0
}

// GetPrivateHostedZoneID indicates an expected call of GetPrivateHostedZoneID
func (mr *MockAWSMockRecorder) GetPrivateHostedZoneID() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetPrivateHostedZoneID", reflect.TypeOf((*MockAWS)(nil).GetPrivateHostedZoneID))
}

// GetPublicHostedZoneNames mocks base method
func (m *MockAWS) GetPublicHostedZoneNames() []string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetPublicHostedZoneNames")
	ret0, _ := ret[0].([]string)
	return ret0
}

// GetPublicHostedZoneNames indicates an expected call of GetPublicHostedZoneNames
func (mr *MockAWSMockRecorder) GetPublicHostedZoneNames() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetPublicHostedZoneNames", reflect.TypeOf((*MockAWS)(nil).GetPublicHostedZoneNames))
}

// GetTagByKeyAndZoneID mocks base method
func (m *MockAWS) GetTagByKeyAndZoneID(key, id string, logger logrus.FieldLogger) (*aws.Tag, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetTagByKeyAndZoneID", key, id, logger)
	ret0, _ := ret[0].(*aws.Tag)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetTagByKeyAndZoneID indicates an expected call of GetTagByKeyAndZoneID
func (mr *MockAWSMockRecorder) GetTagByKeyAndZoneID(key, id, logger interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetTagByKeyAndZoneID", reflect.TypeOf((*MockAWS)(nil).GetTagByKeyAndZoneID), key, id, logger)
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

// CreatePublicCNAME mocks base method
func (m *MockAWS) CreatePublicCNAME(dnsName string, dnsEndpoints []string, dnsIdentifier string, logger logrus.FieldLogger) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreatePublicCNAME", dnsName, dnsEndpoints, dnsIdentifier, logger)
	ret0, _ := ret[0].(error)
	return ret0
}

// CreatePublicCNAME indicates an expected call of CreatePublicCNAME
func (mr *MockAWSMockRecorder) CreatePublicCNAME(dnsName, dnsEndpoints, dnsIdentifier, logger interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreatePublicCNAME", reflect.TypeOf((*MockAWS)(nil).CreatePublicCNAME), dnsName, dnsEndpoints, dnsIdentifier, logger)
}

// UpdatePublicRecordIDForCNAME mocks base method
func (m *MockAWS) UpdatePublicRecordIDForCNAME(dnsName, newID string, logger logrus.FieldLogger) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdatePublicRecordIDForCNAME", dnsName, newID, logger)
	ret0, _ := ret[0].(error)
	return ret0
}

// UpdatePublicRecordIDForCNAME indicates an expected call of UpdatePublicRecordIDForCNAME
func (mr *MockAWSMockRecorder) UpdatePublicRecordIDForCNAME(dnsName, newID, logger interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdatePublicRecordIDForCNAME", reflect.TypeOf((*MockAWS)(nil).UpdatePublicRecordIDForCNAME), dnsName, newID, logger)
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

// DeletePublicCNAME mocks base method
func (m *MockAWS) DeletePublicCNAME(dnsName string, logger logrus.FieldLogger) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeletePublicCNAME", dnsName, logger)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeletePublicCNAME indicates an expected call of DeletePublicCNAME
func (mr *MockAWSMockRecorder) DeletePublicCNAME(dnsName, logger interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeletePublicCNAME", reflect.TypeOf((*MockAWS)(nil).DeletePublicCNAME), dnsName, logger)
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

// TagResource mocks base method
func (m *MockAWS) TagResource(resourceID, key, value string, logger logrus.FieldLogger) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "TagResource", resourceID, key, value, logger)
	ret0, _ := ret[0].(error)
	return ret0
}

// TagResource indicates an expected call of TagResource
func (mr *MockAWSMockRecorder) TagResource(resourceID, key, value, logger interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "TagResource", reflect.TypeOf((*MockAWS)(nil).TagResource), resourceID, key, value, logger)
}

// UntagResource mocks base method
func (m *MockAWS) UntagResource(resourceID, key, value string, logger logrus.FieldLogger) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UntagResource", resourceID, key, value, logger)
	ret0, _ := ret[0].(error)
	return ret0
}

// UntagResource indicates an expected call of UntagResource
func (mr *MockAWSMockRecorder) UntagResource(resourceID, key, value, logger interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UntagResource", reflect.TypeOf((*MockAWS)(nil).UntagResource), resourceID, key, value, logger)
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

// DynamoDBEnsureTableDeleted mocks base method
func (m *MockAWS) DynamoDBEnsureTableDeleted(tableName string, logger logrus.FieldLogger) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DynamoDBEnsureTableDeleted", tableName, logger)
	ret0, _ := ret[0].(error)
	return ret0
}

// DynamoDBEnsureTableDeleted indicates an expected call of DynamoDBEnsureTableDeleted
func (mr *MockAWSMockRecorder) DynamoDBEnsureTableDeleted(tableName, logger interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DynamoDBEnsureTableDeleted", reflect.TypeOf((*MockAWS)(nil).DynamoDBEnsureTableDeleted), tableName, logger)
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
func (m *MockAWS) S3LargeCopy(srcBucketName, srcKey, destBucketName, destKey *string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "S3LargeCopy", srcBucketName, srcKey, destBucketName, destKey)
	ret0, _ := ret[0].(error)
	return ret0
}

// S3LargeCopy indicates an expected call of S3LargeCopy
func (mr *MockAWSMockRecorder) S3LargeCopy(srcBucketName, srcKey, destBucketName, destKey interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "S3LargeCopy", reflect.TypeOf((*MockAWS)(nil).S3LargeCopy), srcBucketName, srcKey, destBucketName, destKey)
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

// GetVpcResourcesByVpcID mocks base method
func (m *MockAWS) GetVpcResourcesByVpcID(vpcID string, logger logrus.FieldLogger) (model.ClusterResources, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetVpcResourcesByVpcID", vpcID, logger)
	ret0, _ := ret[0].(model.ClusterResources)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetVpcResourcesByVpcID indicates an expected call of GetVpcResourcesByVpcID
func (mr *MockAWSMockRecorder) GetVpcResourcesByVpcID(vpcID, logger interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetVpcResourcesByVpcID", reflect.TypeOf((*MockAWS)(nil).GetVpcResourcesByVpcID), vpcID, logger)
}

// TagResourcesByCluster mocks base method
func (m *MockAWS) TagResourcesByCluster(clusterResources model.ClusterResources, cluster *model.Cluster, owner string, logger logrus.FieldLogger) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "TagResourcesByCluster", clusterResources, cluster, owner, logger)
	ret0, _ := ret[0].(error)
	return ret0
}

// TagResourcesByCluster indicates an expected call of TagResourcesByCluster
func (mr *MockAWSMockRecorder) TagResourcesByCluster(clusterResources, cluster, owner, logger interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "TagResourcesByCluster", reflect.TypeOf((*MockAWS)(nil).TagResourcesByCluster), clusterResources, cluster, owner, logger)
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

// SecretsManagerValidateExternalDatabaseSecret mocks base method
func (m *MockAWS) SecretsManagerValidateExternalDatabaseSecret(name string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SecretsManagerValidateExternalDatabaseSecret", name)
	ret0, _ := ret[0].(error)
	return ret0
}

// SecretsManagerValidateExternalDatabaseSecret indicates an expected call of SecretsManagerValidateExternalDatabaseSecret
func (mr *MockAWSMockRecorder) SecretsManagerValidateExternalDatabaseSecret(name interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SecretsManagerValidateExternalDatabaseSecret", reflect.TypeOf((*MockAWS)(nil).SecretsManagerValidateExternalDatabaseSecret), name)
}

// SwitchClusterTags mocks base method
func (m *MockAWS) SwitchClusterTags(clusterID, targetClusterID string, logger logrus.FieldLogger) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SwitchClusterTags", clusterID, targetClusterID, logger)
	ret0, _ := ret[0].(error)
	return ret0
}

// SwitchClusterTags indicates an expected call of SwitchClusterTags
func (mr *MockAWSMockRecorder) SwitchClusterTags(clusterID, targetClusterID, logger interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SwitchClusterTags", reflect.TypeOf((*MockAWS)(nil).SwitchClusterTags), clusterID, targetClusterID, logger)
}

// EnsureEKSCluster mocks base method
func (m *MockAWS) EnsureEKSCluster(cluster *model.Cluster, eksMetadata model.EKSMetadata) (*types0.Cluster, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "EnsureEKSCluster", cluster, eksMetadata)
	ret0, _ := ret[0].(*types0.Cluster)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// EnsureEKSCluster indicates an expected call of EnsureEKSCluster
func (mr *MockAWSMockRecorder) EnsureEKSCluster(cluster, eksMetadata interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "EnsureEKSCluster", reflect.TypeOf((*MockAWS)(nil).EnsureEKSCluster), cluster, eksMetadata)
}

// EnsureEKSClusterNodeGroups mocks base method
func (m *MockAWS) EnsureEKSClusterNodeGroups(cluster *model.Cluster, eksMetadata model.EKSMetadata) ([]*types0.Nodegroup, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "EnsureEKSClusterNodeGroups", cluster, eksMetadata)
	ret0, _ := ret[0].([]*types0.Nodegroup)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// EnsureEKSClusterNodeGroups indicates an expected call of EnsureEKSClusterNodeGroups
func (mr *MockAWSMockRecorder) EnsureEKSClusterNodeGroups(cluster, eksMetadata interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "EnsureEKSClusterNodeGroups", reflect.TypeOf((*MockAWS)(nil).EnsureEKSClusterNodeGroups), cluster, eksMetadata)
}

// GetEKSCluster mocks base method
func (m *MockAWS) GetEKSCluster(clusterName string) (*types0.Cluster, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetEKSCluster", clusterName)
	ret0, _ := ret[0].(*types0.Cluster)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetEKSCluster indicates an expected call of GetEKSCluster
func (mr *MockAWSMockRecorder) GetEKSCluster(clusterName interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetEKSCluster", reflect.TypeOf((*MockAWS)(nil).GetEKSCluster), clusterName)
}

// IsClusterReady mocks base method
func (m *MockAWS) IsClusterReady(clusterName string) (bool, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "IsClusterReady", clusterName)
	ret0, _ := ret[0].(bool)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// IsClusterReady indicates an expected call of IsClusterReady
func (mr *MockAWSMockRecorder) IsClusterReady(clusterName interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "IsClusterReady", reflect.TypeOf((*MockAWS)(nil).IsClusterReady), clusterName)
}

// EnsureNodeGroupsDeleted mocks base method
func (m *MockAWS) EnsureNodeGroupsDeleted(cluster *model.Cluster) (bool, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "EnsureNodeGroupsDeleted", cluster)
	ret0, _ := ret[0].(bool)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// EnsureNodeGroupsDeleted indicates an expected call of EnsureNodeGroupsDeleted
func (mr *MockAWSMockRecorder) EnsureNodeGroupsDeleted(cluster interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "EnsureNodeGroupsDeleted", reflect.TypeOf((*MockAWS)(nil).EnsureNodeGroupsDeleted), cluster)
}

// EnsureEKSClusterDeleted mocks base method
func (m *MockAWS) EnsureEKSClusterDeleted(cluster *model.Cluster) (bool, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "EnsureEKSClusterDeleted", cluster)
	ret0, _ := ret[0].(bool)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// EnsureEKSClusterDeleted indicates an expected call of EnsureEKSClusterDeleted
func (mr *MockAWSMockRecorder) EnsureEKSClusterDeleted(cluster interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "EnsureEKSClusterDeleted", reflect.TypeOf((*MockAWS)(nil).EnsureEKSClusterDeleted), cluster)
}

// InstallEKSEBSAddon mocks base method
func (m *MockAWS) InstallEKSEBSAddon(cluster *model.Cluster) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "InstallEKSEBSAddon", cluster)
	ret0, _ := ret[0].(error)
	return ret0
}

// InstallEKSEBSAddon indicates an expected call of InstallEKSEBSAddon
func (mr *MockAWSMockRecorder) InstallEKSEBSAddon(cluster interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "InstallEKSEBSAddon", reflect.TypeOf((*MockAWS)(nil).InstallEKSEBSAddon), cluster)
}

// EnsureLaunchTemplate mocks base method
func (m *MockAWS) EnsureLaunchTemplate(clusterName string, eksMetadata model.EKSMetadata) (*types.LaunchTemplate, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "EnsureLaunchTemplate", clusterName, eksMetadata)
	ret0, _ := ret[0].(*types.LaunchTemplate)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// EnsureLaunchTemplate indicates an expected call of EnsureLaunchTemplate
func (mr *MockAWSMockRecorder) EnsureLaunchTemplate(clusterName, eksMetadata interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "EnsureLaunchTemplate", reflect.TypeOf((*MockAWS)(nil).EnsureLaunchTemplate), clusterName, eksMetadata)
}

// EnsureLaunchTemplateDeleted mocks base method
func (m *MockAWS) EnsureLaunchTemplateDeleted(clusterName string) (bool, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "EnsureLaunchTemplateDeleted", clusterName)
	ret0, _ := ret[0].(bool)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// EnsureLaunchTemplateDeleted indicates an expected call of EnsureLaunchTemplateDeleted
func (mr *MockAWSMockRecorder) EnsureLaunchTemplateDeleted(clusterName interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "EnsureLaunchTemplateDeleted", reflect.TypeOf((*MockAWS)(nil).EnsureLaunchTemplateDeleted), clusterName)
}

// AllowEKSPostgresTraffic mocks base method
func (m *MockAWS) AllowEKSPostgresTraffic(cluster *model.Cluster, eksMetadata model.EKSMetadata) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "AllowEKSPostgresTraffic", cluster, eksMetadata)
	ret0, _ := ret[0].(error)
	return ret0
}

// AllowEKSPostgresTraffic indicates an expected call of AllowEKSPostgresTraffic
func (mr *MockAWSMockRecorder) AllowEKSPostgresTraffic(cluster, eksMetadata interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "AllowEKSPostgresTraffic", reflect.TypeOf((*MockAWS)(nil).AllowEKSPostgresTraffic), cluster, eksMetadata)
}

// RevokeEKSPostgresTraffic mocks base method
func (m *MockAWS) RevokeEKSPostgresTraffic(cluster *model.Cluster, eksMetadata model.EKSMetadata) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "RevokeEKSPostgresTraffic", cluster, eksMetadata)
	ret0, _ := ret[0].(error)
	return ret0
}

// RevokeEKSPostgresTraffic indicates an expected call of RevokeEKSPostgresTraffic
func (mr *MockAWSMockRecorder) RevokeEKSPostgresTraffic(cluster, eksMetadata interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "RevokeEKSPostgresTraffic", reflect.TypeOf((*MockAWS)(nil).RevokeEKSPostgresTraffic), cluster, eksMetadata)
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
