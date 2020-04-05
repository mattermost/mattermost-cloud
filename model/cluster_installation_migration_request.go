package model

import (
	"encoding/json"
	"io"

	"github.com/pkg/errors"
)

// CreateClusterInstallationMigrationRequest specifies the parameters for migrating a cluster installation.
type CreateClusterInstallationMigrationRequest struct {
	ClusterID             string `json:"provider,omitempty"`
	ClusterInstallationID string `json:"version,omitempty"`
}

// NewCreateClusterInstallationMigrationRequestFromReader will create a CreateMigrationRequest from an
// io.Reader with JSON data.
func NewCreateClusterInstallationMigrationRequestFromReader(reader io.Reader) (*CreateClusterInstallationMigrationRequest, error) {
	var createClusterInstallationMigrationRequest CreateClusterInstallationMigrationRequest
	err := json.NewDecoder(reader).Decode(&createClusterInstallationMigrationRequest)
	if err != nil && err != io.EOF {
		return nil, errors.Wrap(err, "failed to decode create cluster installation migration request")
	}

	return &createClusterInstallationMigrationRequest, nil
}
