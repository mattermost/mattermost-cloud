package model

import (
	"encoding/json"
	"io"
)

const (
	// TypeCluster is the string value that represents a cluster.
	TypeCluster = "cluster"
	// TypeInstallation is the string value that represents an installation.
	TypeInstallation = "installation"
	// TypeClusterInstallation is the string value that represents a cluster
	// installation.
	TypeClusterInstallation = "cluster_installaton"
)

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
	OwnerID        string
	Page           int
	PerPage        int
	IncludeDeleted bool
}

// WebhookPayload is the payload sent in every webhook.
type WebhookPayload struct {
	Timestamp int64  `json:"timestamp"`
	ID        string `json:"id"`
	Type      string `json:"type"`
	NewState  string `json:"new_state"`
	OldState  string `json:"old_state"`
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
