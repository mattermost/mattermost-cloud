// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package model

import (
	"fmt"
	"strings"
)

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
	ServerSnippet        string
	//WhitelistSourceRange []string
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
	if ia.ServerSnippet != "" {
		m["nginx.ingress.kubernetes.io/server-snippet"] = ia.ServerSnippet
	}
	if ia.ServerSnippets != "" {
		m["nginx.org/server-snippets"] = ia.ServerSnippets
	}
	//if len(ia.WhitelistSourceRange) > 0 {
	//	m["nginx.ingress.kubernetes.io/whitelist-source-range"] = strings.Join(ia.WhitelistSourceRange, ",")
	//}

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

//func ConfigureIngressAnnotations(whitelist []string, existingHttpSnippet string, existingServerSnippet string, existingSnippet string) *IngressAnnotations {
//	var serverSnippetBuilder strings.Builder
//	var httpSnippetBuilder strings.Builder
//	var configSnippetBuilder strings.Builder
//
//	// Start with the existing snippet
//	configSnippetBuilder.WriteString(existingSnippet + "\n")
//	httpSnippetBuilder.WriteString(existingHttpSnippet + "\n")
//	serverSnippetBuilder.WriteString(existingServerSnippet + "\n")
//
//	// Use map to set a variable based on client IP
//	httpSnippetBuilder.WriteString("map $remote_addr $ip_access {\n")
//	httpSnippetBuilder.WriteString("    default \"deny\";\n")
//	for _, ip := range whitelist {
//		if ip != "" {
//			httpSnippetBuilder.WriteString(fmt.Sprintf("    %s \"allow\";\n", ip))
//		}
//	}
//	httpSnippetBuilder.WriteString("}\n")
//
//	// Use the variable for conditional error_page directive
//	serverSnippetBuilder.WriteString(`
//       error_page 403 = @custom_403;
//       location @custom_403 {
//           if ($ip_access = "deny") {
//               return 419;
//           }
//           return 403;
//       }
//   `)
//
//	var allowDirectivesBuilder strings.Builder
//	for _, ip := range whitelist {
//		if ip != "" {
//			allowDirectivesBuilder.WriteString(fmt.Sprintf("allow %s;\n", ip))
//		}
//	}
//
//	// Build the configSnippet using strings.Builder
//	configSnippetBuilder.WriteString(allowDirectivesBuilder.String())
//	configSnippetBuilder.WriteString(`
//        deny all;
//    `)
//	configSnippet := configSnippetBuilder.String()
//	httpSnippet := httpSnippetBuilder.String()
//	serverSnippet := serverSnippetBuilder.String()
//
//	// Setting up the IngressAnnotations struct
//	ia := &IngressAnnotations{
//		HttpSnippet:          httpSnippet,
//		ServerSnippet:        serverSnippet,
//		ConfigurationSnippet: configSnippet,
//	}
//	fmt.Println(ia.HttpSnippet)
//	fmt.Println(ia.ServerSnippet)
//	fmt.Println(ia.ConfigurationSnippet)
//	return ia
//}

//func ConfigureIngressAnnotations(whitelist []string, existingSnippet string) *IngressAnnotations {
//	// Create the allow directives for each IP in the whitelist
//	var allowDirectivesBuilder strings.Builder
//	for _, ip := range whitelist {
//		if ip != "" {
//			allowDirectivesBuilder.WriteString(fmt.Sprintf("allow %s;\n", ip))
//		}
//	}
//
//	// Build the configSnippet using strings.Builder
//	configSnippetBuilder := strings.Builder{}
//	configSnippetBuilder.WriteString(existingSnippet)
//	configSnippetBuilder.WriteString("\n")
//	configSnippetBuilder.WriteString(allowDirectivesBuilder.String())
//	configSnippetBuilder.WriteString(`
//        deny all;
//    `)
//	configSnippet := configSnippetBuilder.String()
//
//	// Setting up the IngressAnnotations struct
//	ia := &IngressAnnotations{
//		ConfigurationSnippet: configSnippet,
//	}
//	return ia
//}

func ConfigureIngressAnnotations(whitelist []string, existingSnippet string) *IngressAnnotations {
	// Create the allow directives for each IP in the whitelist
	var allowDirectivesBuilder strings.Builder
	for _, ip := range whitelist {
		if ip != "" {
			allowDirectivesBuilder.WriteString(fmt.Sprintf("allow %s;\n", ip))
		}
	}

	// Construct the if condition for setting the $maintenance variable
	var ifConditionBuilder strings.Builder
	if len(whitelist) > 0 {
		ifConditionBuilder.WriteString("if ($remote_addr ~ (")
		for i, ip := range whitelist {
			ifConditionBuilder.WriteString(ip)
			if i < len(whitelist)-1 {
				ifConditionBuilder.WriteString("|")
			}
		}
		ifConditionBuilder.WriteString(")) {\n    set $reroute off;\n}\n")
	}

	// Build the configSnippet using strings.Builder
	configSnippetBuilder := strings.Builder{}
	configSnippetBuilder.WriteString(existingSnippet)
	configSnippetBuilder.WriteString("\n")
	configSnippetBuilder.WriteString(allowDirectivesBuilder.String())
	configSnippetBuilder.WriteString(ifConditionBuilder.String())
	// Add conditional error_page directive based on $reroute
	configSnippetBuilder.WriteString(`
        if ($reroute != "off") {
            error_page 403 =419 /custom_419_page;
        }
    `)
	configSnippetBuilder.WriteString(`
        deny all;
    `)
	configSnippet := configSnippetBuilder.String()

	// Setting up the IngressAnnotations struct
	ia := &IngressAnnotations{
		ConfigurationSnippet: configSnippet,
	}
	return ia
}
