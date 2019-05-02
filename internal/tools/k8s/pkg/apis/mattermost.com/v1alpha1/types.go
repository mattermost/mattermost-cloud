package v1alpha1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ClusterInstallation is a custom kubernetes resource for a mattermost installation.
type ClusterInstallation struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`
	// +optional
	Spec ClusterInstallationSpec `json:"spec"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ClusterInstallationList is a list of ClusterInstallation resources.
type ClusterInstallationList struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []ClusterInstallation `json:"items"`
}

// ClusterInstallationSpec is the spec object of a ClusterInstallation.
type ClusterInstallationSpec struct {
	IngressName string `json:"ingressName"`
}
