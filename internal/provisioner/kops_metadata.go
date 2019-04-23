package provisioner

import (
	"encoding/json"
)

// KopsMetadata is the provisioner metadata stored in a store.Cluster.
type KopsMetadata struct {
	Name string
}

// NewKopsMetadata creates an instance of KopsMetadata given the raw provisioner metadata.
func NewKopsMetadata(provisionerMetadata []byte) *KopsMetadata {
	kopsMetadata := KopsMetadata{}

	if provisionerMetadata == nil {
		return &kopsMetadata
	}

	_ = json.Unmarshal(provisionerMetadata, &kopsMetadata)

	return &kopsMetadata
}
