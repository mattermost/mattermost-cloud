// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package model

import "strings"

type IngressAnnotations struct {
	TLSACME              string
	ProxyBuffering       string
	ProxyBodySize        string
	ProxySendTimeout     string
	ProxyReadTimeout     string
	ProxyMaxTempFileSize string
	SSLRedirect          string
	ConfigurationSnippet string
	ServerSnippets       string
	WhitelistSourceRange []string
}

func (ia *IngressAnnotations) ToMap() map[string]string {
	m := make(map[string]string)

	if ia == nil {
		return m
	}

	if ia.TLSACME != "" {
		m["kubernetes.io/tls-acme"] = ia.TLSACME
	}
	if ia.ProxyBuffering != "" {
		m["nginx.ingress.kubernetes.io/proxy-buffering"] = ia.ProxyBuffering
	}
	if ia.ProxyBodySize != "" {
		m["nginx.ingress.kubernetes.io/proxy-body-size"] = ia.ProxyBodySize
	}
	if ia.ProxySendTimeout != "" {
		m["nginx.ingress.kubernetes.io/proxy-send-timeout"] = ia.ProxySendTimeout
	}
	if ia.ProxyReadTimeout != "" {
		m["nginx.ingress.kubernetes.io/proxy-read-timeout"] = ia.ProxyReadTimeout
	}
	if ia.ProxyMaxTempFileSize != "" {
		m["nginx.ingress.kubernetes.io/proxy-max-temp-file-size"] = ia.ProxyMaxTempFileSize
	}
	if ia.SSLRedirect != "" {
		m["nginx.ingress.kubernetes.io/ssl-redirect"] = ia.SSLRedirect
	}
	if ia.ConfigurationSnippet != "" {
		m["nginx.ingress.kubernetes.io/configuration-snippet"] = ia.ConfigurationSnippet
	}
	if ia.ServerSnippets != "" {
		m["nginx.org/server-snippets"] = ia.ServerSnippets
	}
	if len(ia.WhitelistSourceRange) > 0 {
		m["nginx.ingress.kubernetes.io/whitelist-source-range"] = strings.Join(ia.WhitelistSourceRange, ",")
	}

	return m
}

func (ia *IngressAnnotations) SetDefaults() {
	if ia.TLSACME == "" {
		ia.TLSACME = "true"
	}
	if ia.ProxyBuffering == "" {
		ia.ProxyBuffering = "on"
	}
	if ia.ProxyBodySize == "" {
		ia.ProxyBodySize = "100m"
	}
	if ia.ProxySendTimeout == "" {
		ia.ProxySendTimeout = "600"
	}
	if ia.ProxyReadTimeout == "" {
		ia.ProxyReadTimeout = "600"
	}
	if ia.ProxyMaxTempFileSize == "" {
		ia.ProxyMaxTempFileSize = "0"
	}
	if ia.SSLRedirect == "" {
		ia.SSLRedirect = "true"
	}
	if ia.ConfigurationSnippet == "" {
		ia.ConfigurationSnippet = `
                  proxy_force_ranges on;
                  add_header Strict-Transport-Security "max-age=31536000; includeSubDomains" always;`
	}
	if ia.ServerSnippets == "" {
		ia.ServerSnippets = "gzip on;"
	}
}

func (ia *IngressAnnotations) SetHibernatingDefaults() {
	ia.ConfigurationSnippet = "return 410;"
}
