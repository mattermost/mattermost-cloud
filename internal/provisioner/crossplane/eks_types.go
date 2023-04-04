// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package crossplane

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var (
	GroupVersion = schema.GroupVersion{Group: "cloud.mattermost.io", Version: "v1alpha1"}
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type EKS struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              EKSSpec `json:"spec"`
}

type EKSMetadata struct {
	Name string `json:"name"`
}

type EKSSpec struct {
	CompositionSelector EKSSpecCompositionSelector `json:"compositionSelector"`
	ID                  string                     `json:"id"`
	Parameters          EKSSpecParameters          `json:"parameters"`
	ResourceConfig      EKSSpecResourceConfig      `json:"resourceConfig"`
}

type EKSSpecCompositionSelector struct {
	MatchLabels EKSSpecCompositionSelectorMatchLabels `json:"matchLabels"`
}

type EKSSpecCompositionSelectorMatchLabels struct {
	Provider string `json:"provider"`
	Service  string `json:"service"`
}

type EKSSpecParameters struct {
	Version               string   `json:"version"`
	AccountID             string   `json:"account_id"`
	Region                string   `json:"region"`
	Environment           string   `json:"environment"`
	ClusterShortName      string   `json:"cluster_short_name"`
	EndpointPrivateAccess bool     `json:"endpoint_private_access"`
	EndpointPublicAccess  bool     `json:"endpoint_public_access"`
	VpcID                 string   `json:"vpc_id"`
	SubnetIds             []string `json:"subnet_ids"`
	PrivateSubnetIds      []string `json:"private_subnet_ids"`
	NodeCount             int      `json:"node_count"`
	InstanceType          string   `json:"instance_type"`
	ImageID               string   `json:"image_id"`
	LaunchTemplateVersion string   `json:"launch_template_version"`
}

type EKSSpecResourceConfig struct {
	ProviderConfigName string `json:"providerConfigName"`
}
