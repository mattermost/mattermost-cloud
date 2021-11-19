// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package model

import (
	"encoding/json"
	"io"
)

// ResourceType specifies a type of Provisioners' resource.
type ResourceType string

const (
	// TypeCluster is the string value that represents a cluster.
	TypeCluster ResourceType = "cluster"
	// TypeInstallation is the string value that represents an installation.
	TypeInstallation ResourceType = "installation"
	// TypeClusterInstallation is the string value that represents a cluster
	// installation.
	TypeClusterInstallation ResourceType = "cluster_installation"
	// TypeInstallationBackup is the string value that represents an installation backup.
	TypeInstallationBackup ResourceType = "installation_backup"
	// TypeInstallationDBRestoration is the string value that represents an installation db restoration operation.
	TypeInstallationDBRestoration ResourceType = "installation_db_restoration_operation"
	// TypeInstallationDBMigration is the string value that represents an installation db migration operation.
	TypeInstallationDBMigration ResourceType = "installation_db_migration_operation"
)

// String converts ResourceType to string.
func (t ResourceType) String() string {
	return string(t)
}

// Webhook is
type Webhook struct {
	ID       string
	OwnerID  string
	URL      string
	CreateAt int64
	DeleteAt int64
}

// WebhookFilter describes the parameters used to constrain a set of webhooks.
type WebhookFilter struct {
	Paging
	OwnerID string
}

// WebhookPayload is the payload sent in every webhook.
type WebhookPayload struct {
	EventID   string            `json:"event_id"`
	Timestamp int64             `json:"timestamp"`
	ID        string            `json:"id"`
	Type      ResourceType      `json:"type"`
	NewState  string            `json:"new_state"`
	OldState  string            `json:"old_state"`
	ExtraData map[string]string `json:"extra_data,omitempty"`
}

// IsDeleted returns whether the webhook was marked as deleted or not.
func (w *Webhook) IsDeleted() bool {
	return w.DeleteAt != 0
}

// ToJSON returns a JSON string representation of the webhook payload.
func (p *WebhookPayload) ToJSON() (string, error) {
	b, err := json.Marshal(p)
	if err != nil {
		return "", err
	}

	return string(b), nil
}

// WebhookFromReader decodes a json-encoded webhook from the given io.Reader.
func WebhookFromReader(reader io.Reader) (*Webhook, error) {
	webhook := Webhook{}
	decoder := json.NewDecoder(reader)
	err := decoder.Decode(&webhook)
	if err != nil && err != io.EOF {
		return nil, err
	}

	return &webhook, nil
}

// WebhooksFromReader decodes a json-encoded list of webhooks from the given io.Reader.
func WebhooksFromReader(reader io.Reader) ([]*Webhook, error) {
	webhooks := []*Webhook{}
	decoder := json.NewDecoder(reader)

	err := decoder.Decode(&webhooks)
	if err != nil && err != io.EOF {
		return nil, err
	}

	return webhooks, nil
}

// WebhookPayloadFromReader decodes a json-encoded webhook payload from the given io.Reader.
func WebhookPayloadFromReader(reader io.Reader) (*WebhookPayload, error) {
	payload := WebhookPayload{}
	decoder := json.NewDecoder(reader)
	err := decoder.Decode(&payload)
	if err != nil && err != io.EOF {
		return nil, err
	}

	return &payload, nil
}
