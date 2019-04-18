package provisioner

import (
	"encoding/json"
	"fmt"

	"github.com/mattermost/mattermost-cloud/internal/store"
	"github.com/mattermost/mattermost-server/model"
)

// KopsMetadata is the provisioner metadata stored in a store.Cluster.
type KopsMetadata struct {
	Name string
}

// NewKopsMetadata creates an instance of KopsMetadata given the raw provisioner metadata.
func NewKopsMetadata(provisionerMetadata []byte) *KopsMetadata {
	kopsMetadata := KopsMetadata{}
	_ = json.Unmarshal(provisionerMetadata, &kopsMetadata)

	return &kopsMetadata
}

// KopsCluster is a store.Cluster with knowledge of how to decode the kops provisioner metadata.
type KopsCluster struct {
	store.Cluster
	kopsMetadata *KopsMetadata
}

// NewKopsCluster creates a new KopsCluster.
func NewKopsCluster(provider string) *KopsCluster {
	id := model.NewId()
	kopsCluster := &KopsCluster{
		Cluster: store.Cluster{
			ID:          id,
			Provider:    provider,
			Provisioner: "kops",
		},
	}
	kopsCluster.SetKopsMetadata(KopsMetadata{
		Name: fmt.Sprintf("%s-kops.k8s.local", id),
	})

	return kopsCluster
}

// KopsClusterFromCluster creates a new KopsCluster from a store.Cluster.
func KopsClusterFromCluster(cluster *store.Cluster) *KopsCluster {
	kopsCluster := &KopsCluster{
		Cluster:      *cluster,
		kopsMetadata: NewKopsMetadata(cluster.ProvisionerMetadata),
	}

	return kopsCluster
}

// GetKopsMetadata returns the decoded KopsMetadata from the provisioner metadata.
func (kopsCluster *KopsCluster) GetKopsMetadata() KopsMetadata {
	return *kopsCluster.kopsMetadata
}

// SetKopsMetadata updates the encoded provisioner metadata from the given KopsMetadata.
func (kopsCluster *KopsCluster) SetKopsMetadata(kopsMetadata KopsMetadata) {
	kopsCluster.kopsMetadata = &kopsMetadata
	provisionerMetadata, _ := json.Marshal(kopsMetadata)
	kopsCluster.ProvisionerMetadata = provisionerMetadata
}

// KopsName returns the name given to kops when the cluster was created.
func (kopsCluster *KopsCluster) KopsName() string {
	return kopsCluster.kopsMetadata.Name
}
