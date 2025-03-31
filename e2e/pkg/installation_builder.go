// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

//go:build e2e
// +build e2e

package pkg

import "github.com/mattermost/mattermost-cloud/model"

// InstallationBuilder is a helper to create CreateInstallationRequest.
type InstallationBuilder struct {
	request *model.CreateInstallationRequest
}

// NewInstallationBuilderWithDefaults sets up new InstallationBuilder with reasonable defaults.
func NewInstallationBuilderWithDefaults() *InstallationBuilder {
	return &InstallationBuilder{request: &model.CreateInstallationRequest{
		OwnerID:   "e2e-test",
		Version:   "10.6.1",
		Image:     "mattermost/mattermost-enterprise-edition",
		Size:      "1000users",
		Affinity:  "multitenant",
		Database:  model.InstallationDatabaseMultiTenantRDSPostgres,
		Filestore: model.InstallationFilestoreBifrost,
	}}
}

// DB sets database type used for Installation.
func (b *InstallationBuilder) DB(db string) *InstallationBuilder {
	b.request.Database = db
	return b
}

// FileStore sets file store type used for Installation.
func (b *InstallationBuilder) FileStore(fileStore string) *InstallationBuilder {
	b.request.Filestore = fileStore
	return b
}

// DNS sets DNS for Installation.
func (b *InstallationBuilder) DNS(dns string) *InstallationBuilder {
	b.request.DNS = dns
	return b
}

// Version sets Mattermost version used for Installation.
func (b *InstallationBuilder) Version(version string) *InstallationBuilder {
	b.request.Version = version
	return b
}

// Owner sets Installation's owner.
func (b *InstallationBuilder) Owner(owner string) *InstallationBuilder {
	b.request.OwnerID = owner
	return b
}

// Group sets Installation's group.
func (b *InstallationBuilder) Group(group string) *InstallationBuilder {
	b.request.GroupID = group
	return b
}

// Size sets Installation's size.
func (b *InstallationBuilder) Size(size string) *InstallationBuilder {
	b.request.Size = size
	return b
}

// Annotations sets Installation's annotations.
func (b *InstallationBuilder) Annotations(annotations []string) *InstallationBuilder {
	b.request.Annotations = annotations
	return b
}

// CreateRequest returns CreateInstallationRequest based on the builder.
func (b *InstallationBuilder) CreateRequest() *model.CreateInstallationRequest {
	return b.request
}
