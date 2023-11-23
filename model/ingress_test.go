package model_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mattermost/mattermost-cloud/model"
)

func TestIngressAnnotationsToMap(t *testing.T) {
	t.Run("returns empty map when IngressAnnotations is nil", func(t *testing.T) {
		var ia *model.IngressAnnotations
		expected := make(map[string]string)

		actual := ia.ToMap()

		require.Equal(t, expected, actual)
	})

	t.Run("returns expected map when IngressAnnotations is not nil", func(t *testing.T) {
		ia := &model.IngressAnnotations{
			TLSACME:              "true",
			ProxyBuffering:       "on",
			ProxyBodySize:        "100m",
			ProxySendTimeout:     "600",
			ProxyReadTimeout:     "600",
			ProxyMaxTempFileSize: "0",
			SSLRedirect:          "true",
			ConfigurationSnippet: `
                  proxy_force_ranges on;
                  add_header Strict-Transport-Security "max-age=31536000; includeSubDomains" always;`,
			ServerSnippets: "gzip on;",
		}
		expected := map[string]string{
			"kubernetes.io/tls-acme":                               "true",
			"nginx.ingress.kubernetes.io/proxy-buffering":          "on",
			"nginx.ingress.kubernetes.io/proxy-body-size":          "100m",
			"nginx.ingress.kubernetes.io/proxy-send-timeout":       "600",
			"nginx.ingress.kubernetes.io/proxy-read-timeout":       "600",
			"nginx.ingress.kubernetes.io/proxy-max-temp-file-size": "0",
			"nginx.ingress.kubernetes.io/ssl-redirect":             "true",
			"nginx.ingress.kubernetes.io/configuration-snippet": `
                  proxy_force_ranges on;
                  add_header Strict-Transport-Security "max-age=31536000; includeSubDomains" always;`,
			"nginx.org/server-snippets":                          "gzip on;",
			"nginx.ingress.kubernetes.io/whitelist-source-range": "192.168.0.1/24,10.0.0.1/16",
		}

		actual := ia.ToMap()

		require.Equal(t, expected, actual)
	})

	t.Run("returns expected map when some fields are empty", func(t *testing.T) {
		ia := &model.IngressAnnotations{
			TLSACME:              "",
			ProxyBuffering:       "",
			ProxyBodySize:        "",
			ProxySendTimeout:     "",
			ProxyReadTimeout:     "",
			ProxyMaxTempFileSize: "",
			SSLRedirect:          "",
			ConfigurationSnippet: "",
			ServerSnippets:       "",
		}
		expected := map[string]string{}

		actual := ia.ToMap()

		require.Equal(t, expected, actual)
	})

	t.Run("returns expected map when some fields are not empty", func(t *testing.T) {
		ia := &model.IngressAnnotations{
			TLSACME:              "",
			ProxyBuffering:       "on",
			ProxyBodySize:        "",
			ProxySendTimeout:     "600",
			ProxyReadTimeout:     "",
			ProxyMaxTempFileSize: "",
			SSLRedirect:          "",
			ConfigurationSnippet: `
                  proxy_force_ranges on;
                  add_header Strict-Transport-Security "max-age=31536000; includeSubDomains" always;`,
			ServerSnippets: "",
		}
		expected := map[string]string{
			"nginx.ingress.kubernetes.io/proxy-buffering":    "on",
			"nginx.ingress.kubernetes.io/proxy-send-timeout": "600",
			"nginx.ingress.kubernetes.io/configuration-snippet": `
                  proxy_force_ranges on;
                  add_header Strict-Transport-Security "max-age=31536000; includeSubDomains" always;`,
			"nginx.ingress.kubernetes.io/whitelist-source-range": "192.168.0.1/24,10.0.0.1/16",
		}

		actual := ia.ToMap()

		require.Equal(t, expected, actual)
	})
}

func TestIngressAnnotationsSetDefaults(t *testing.T) {
	t.Run("sets default values when fields are empty", func(t *testing.T) {
		ia := &model.IngressAnnotations{}
		expected := &model.IngressAnnotations{
			TLSACME:              "true",
			ProxyBuffering:       "on",
			ProxyBodySize:        "100m",
			ProxySendTimeout:     "600",
			ProxyReadTimeout:     "600",
			ProxyMaxTempFileSize: "0",
			SSLRedirect:          "true",
			ConfigurationSnippet: `
                  proxy_force_ranges on;
                  add_header Strict-Transport-Security "max-age=31536000; includeSubDomains" always;`,
			ServerSnippets: "gzip on;",
		}

		ia.SetDefaults()

		require.Equal(t, expected, ia)
	})

	t.Run("does not overwrite non-empty fields", func(t *testing.T) {
		ia := &model.IngressAnnotations{
			TLSACME:              "false",
			ProxyBuffering:       "off",
			ProxyBodySize:        "50m",
			ProxySendTimeout:     "300",
			ProxyReadTimeout:     "300",
			ProxyMaxTempFileSize: "10m",
			SSLRedirect:          "false",
			ConfigurationSnippet: "return 404;",
			ServerSnippets:       "gzip off;",
		}
		expected := &model.IngressAnnotations{
			TLSACME:              "false",
			ProxyBuffering:       "off",
			ProxyBodySize:        "50m",
			ProxySendTimeout:     "300",
			ProxyReadTimeout:     "300",
			ProxyMaxTempFileSize: "10m",
			SSLRedirect:          "false",
			ConfigurationSnippet: "return 404;",
			ServerSnippets:       "gzip off;",
		}

		ia.SetDefaults()

		require.Equal(t, expected, ia)
	})
}

func TestIngressAnnotationsSetHibernatingDefaults(t *testing.T) {
	t.Run("sets ConfigurationSnippet to 'return 410;'", func(t *testing.T) {
		ia := &model.IngressAnnotations{
			ConfigurationSnippet: "return 404;",
		}
		expected := &model.IngressAnnotations{
			ConfigurationSnippet: "return 410;",
		}

		ia.SetHibernatingDefaults()

		require.Equal(t, expected, ia)
	})
}

func TestConfigureIngressWithServerSnippets(t *testing.T) {
	model.ConfigureIngressAnnotations([]string{"192.168.0.1/32"}, "", "", "")
}
