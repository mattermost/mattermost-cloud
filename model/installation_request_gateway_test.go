// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package model_test

import (
	"bytes"
	"testing"

	"github.com/mattermost/mattermost-cloud/model"
	"github.com/stretchr/testify/require"
)

func newHTTPRouteCreateRequest(gatewayConfig *model.GatewayConfig) *model.CreateInstallationRequest {
	return &model.CreateInstallationRequest{
		Name:          "httproute",
		OwnerID:       "owner1",
		DNS:           "httproute.example.com",
		IngressType:   model.InstallationIngressHTTPRoute,
		GatewayConfig: gatewayConfig,
	}
}

func TestCreateInstallationRequestHTTPRouteSupport(t *testing.T) {
	model.SetDeployOperators(true, true)
	defer model.SetHTTPRouteSupported(false)

	t.Run("rejected when not supported", func(t *testing.T) {
		model.SetHTTPRouteSupported(false)
		request := newHTTPRouteCreateRequest(&model.GatewayConfig{Name: "cilium-gateway"})
		request.SetDefaults()
		require.Error(t, request.Validate())
	})

	t.Run("accepted when supported", func(t *testing.T) {
		model.SetHTTPRouteSupported(true)
		request := newHTTPRouteCreateRequest(&model.GatewayConfig{Name: "cilium-gateway"})
		request.SetDefaults()
		require.NoError(t, request.Validate())
	})

	t.Run("ingress unaffected when not supported", func(t *testing.T) {
		model.SetHTTPRouteSupported(false)
		request := newHTTPRouteCreateRequest(nil)
		request.IngressType = model.InstallationIngressIngress
		request.SetDefaults()
		require.NoError(t, request.Validate())
	})
}

func TestCreateInstallationRequestGatewayDefaults(t *testing.T) {
	model.SetDefaultGatewayConfig("cilium-gateway", "kube-system", "https")
	defer model.SetDefaultGatewayConfig("", "", "")

	defaults := model.GatewayConfig{Name: "cilium-gateway", Namespace: "kube-system", SectionName: "https"}

	t.Run("nil config gets a copy of the default", func(t *testing.T) {
		request := newHTTPRouteCreateRequest(nil)
		request.SetDefaults()
		require.Equal(t, &defaults, request.GatewayConfig)

		// Changing one request's config must not change the server default.
		request.GatewayConfig.Namespace = "changed"
		other := newHTTPRouteCreateRequest(nil)
		other.SetDefaults()
		require.Equal(t, &defaults, other.GatewayConfig)
	})

	t.Run("default gateway name fills empty fields", func(t *testing.T) {
		request := newHTTPRouteCreateRequest(&model.GatewayConfig{Name: "cilium-gateway"})
		request.SetDefaults()
		require.Equal(t, &defaults, request.GatewayConfig)
	})

	t.Run("default gateway name keeps explicit fields", func(t *testing.T) {
		request := newHTTPRouteCreateRequest(&model.GatewayConfig{Name: "cilium-gateway", Namespace: "gateways"})
		request.SetDefaults()
		require.Equal(t, &model.GatewayConfig{Name: "cilium-gateway", Namespace: "gateways", SectionName: "https"}, request.GatewayConfig)
	})

	t.Run("other gateway name is kept as is", func(t *testing.T) {
		request := newHTTPRouteCreateRequest(&model.GatewayConfig{Name: "other-gateway"})
		request.SetDefaults()
		require.Equal(t, &model.GatewayConfig{Name: "other-gateway"}, request.GatewayConfig)
	})

	t.Run("ingress requests get no gateway config", func(t *testing.T) {
		request := newHTTPRouteCreateRequest(nil)
		request.IngressType = model.InstallationIngressIngress
		request.SetDefaults()
		require.Nil(t, request.GatewayConfig)
	})
}

func TestPatchInstallationRequestHTTPRoute(t *testing.T) {
	model.SetDefaultGatewayConfig("cilium-gateway", "kube-system", "https")
	defer model.SetDefaultGatewayConfig("", "", "")
	defer model.SetHTTPRouteSupported(false)

	t.Run("rejected when not supported", func(t *testing.T) {
		model.SetHTTPRouteSupported(false)
		_, err := model.NewPatchInstallationRequestFromReader(bytes.NewReader([]byte(
			`{"IngressType":"httproute","GatewayConfig":{"Name":"cilium-gateway"}}`)))
		require.Error(t, err)
	})

	t.Run("switch to httproute uses the default gateway", func(t *testing.T) {
		model.SetHTTPRouteSupported(true)
		patch, err := model.NewPatchInstallationRequestFromReader(bytes.NewReader([]byte(
			`{"IngressType":"httproute"}`)))
		require.NoError(t, err)

		installation := &model.Installation{IngressType: model.InstallationIngressIngress}
		require.True(t, patch.Apply(installation))
		require.Equal(t, model.InstallationIngressHTTPRoute, installation.IngressType)
		require.Equal(t, &model.GatewayConfig{Name: "cilium-gateway", Namespace: "kube-system", SectionName: "https"}, installation.GatewayConfig)
	})

	t.Run("httproute patch keeps the stored gateway", func(t *testing.T) {
		model.SetHTTPRouteSupported(true)
		patch, err := model.NewPatchInstallationRequestFromReader(bytes.NewReader([]byte(
			`{"IngressType":"httproute"}`)))
		require.NoError(t, err)

		custom := &model.GatewayConfig{Name: "custom-gateway", Namespace: "gateways", SectionName: "web"}
		installation := &model.Installation{IngressType: model.InstallationIngressHTTPRoute, GatewayConfig: custom}
		require.False(t, patch.Apply(installation))
		require.Equal(t, &model.GatewayConfig{Name: "custom-gateway", Namespace: "gateways", SectionName: "web"}, installation.GatewayConfig)
	})

	t.Run("switch to httproute without any default gateway", func(t *testing.T) {
		model.SetHTTPRouteSupported(true)
		model.SetDefaultGatewayConfig("", "", "")
		defer model.SetDefaultGatewayConfig("cilium-gateway", "kube-system", "https")
		_, err := model.NewPatchInstallationRequestFromReader(bytes.NewReader([]byte(
			`{"IngressType":"httproute"}`)))
		require.Error(t, err)
	})

	t.Run("partial default gateway is filled", func(t *testing.T) {
		model.SetHTTPRouteSupported(true)
		patch, err := model.NewPatchInstallationRequestFromReader(bytes.NewReader([]byte(
			`{"GatewayConfig":{"Name":"cilium-gateway"}}`)))
		require.NoError(t, err)
		require.Equal(t, &model.GatewayConfig{Name: "cilium-gateway", Namespace: "kube-system", SectionName: "https"}, patch.GatewayConfig)
	})

	t.Run("patch without ingress changes is untouched", func(t *testing.T) {
		patch, err := model.NewPatchInstallationRequestFromReader(bytes.NewReader([]byte(`{"Version":"11.0"}`)))
		require.NoError(t, err)
		require.Nil(t, patch.GatewayConfig)
	})
}
