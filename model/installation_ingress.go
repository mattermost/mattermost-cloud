// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package model

import (
	"database/sql/driver"
	"encoding/json"

	"github.com/pkg/errors"
)

const (
	// InstallationIngressIngress routes traffic via a Kubernetes Ingress resource.
	InstallationIngressIngress = "ingress"
	// InstallationIngressHTTPRoute routes traffic via a Gateway API HTTPRoute resource.
	InstallationIngressHTTPRoute = "httproute"
)

// IsSupportedIngressType returns true if the given ingress type is valid.
func IsSupportedIngressType(t string) bool {
	return t == InstallationIngressIngress || t == InstallationIngressHTTPRoute
}

// GatewayConfig holds the reference to a Gateway resource for HTTPRoute-based routing.
type GatewayConfig struct {
	// Name is the name of the Gateway resource. Required when IngressType is "httproute".
	Name string
	// Namespace is the namespace of the Gateway. Defaults to the installation namespace.
	Namespace string `json:"Namespace,omitempty"`
	// SectionName targets a specific listener on the Gateway.
	SectionName string `json:"SectionName,omitempty"`
}

// Value implements the driver.Valuer interface for database storage.
func (g *GatewayConfig) Value() (driver.Value, error) {
	if g == nil {
		return nil, nil
	}
	return json.Marshal(g)
}

// Scan implements the sql.Scanner interface for database retrieval.
func (g *GatewayConfig) Scan(src interface{}) error {
	if src == nil {
		return nil
	}
	source, ok := src.([]byte)
	if !ok {
		return errors.New("could not assert type of GatewayConfig")
	}
	return json.Unmarshal(source, g)
}
