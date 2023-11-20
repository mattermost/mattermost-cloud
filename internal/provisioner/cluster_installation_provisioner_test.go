// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package provisioner

import (
	"fmt"
	"github.com/mattermost/mattermost-cloud/model"
	"reflect"
	"strings"
)
import "testing"

func TestAddSourceRangeWhitelistToAnnotations_WhitelistLogic(t *testing.T) {
	annotations := &model.IngressAnnotations{}
	allowedIPRanges := &model.AllowedIPRanges{
		{CIDRBlock: "123.45.67.89/32", Enabled: true},
		{CIDRBlock: "98.76.54.32/32", Enabled: true},
	}
	internalIPRanges := []string{"10.0.0.0/16"}

	addSourceRangeWhitelistToAnnotations(annotations, allowedIPRanges, internalIPRanges)

	// Check if WhitelistSourceRange is correctly populated
	expectedIPs := []string{"123.45.67.89/32", "98.76.54.32/32", "10.0.0.0/16"}
	if !reflect.DeepEqual(annotations.WhitelistSourceRange, expectedIPs) {
		t.Errorf("Expected WhitelistSourceRange to be %v, got %v", expectedIPs, annotations.WhitelistSourceRange)
	}

	// Check if ConfigurationSnippet contains the correct 'allow' directives
	for _, ip := range expectedIPs {
		if !strings.Contains(annotations.ConfigurationSnippet, fmt.Sprintf("allow %s;", ip)) {
			t.Errorf("ConfigurationSnippet missing allow directive for %s", ip)
		}
	}

	// Check if the custom logic for handling a 419 status code is correctly added
	expectedErrorPageDirective := "error_page 403 =419 /custom_419_page;"
	if !strings.Contains(annotations.ConfigurationSnippet, expectedErrorPageDirective) {
		t.Errorf("Expected to find '%s' in ConfigurationSnippet, got %s", expectedErrorPageDirective, annotations.ConfigurationSnippet)
	}
}

func TestAddSourceRangeWhitelistToAnnotations_ExistingConfigPreservation(t *testing.T) {
	existingConfig := "proxy_buffer_size 128k;"
	annotations := &model.IngressAnnotations{ConfigurationSnippet: existingConfig}
	allowedIPRanges := &model.AllowedIPRanges{
		{CIDRBlock: "123.45.67.89/32", Enabled: true},
	}
	internalIPRanges := []string{"10.0.0.0/16"}

	addSourceRangeWhitelistToAnnotations(annotations, allowedIPRanges, internalIPRanges)

	// Check if the existing configuration is preserved in ConfigurationSnippet
	if !strings.Contains(annotations.ConfigurationSnippet, existingConfig) {
		t.Errorf("Existing configuration was not preserved in ConfigurationSnippet")
	}
}
func TestAddSourceRangeWhitelistToAnnotations_AllRulesDisabled(t *testing.T) {
	annotations := &model.IngressAnnotations{}
	allowedIPRanges := &model.AllowedIPRanges{
		{CIDRBlock: "123.45.67.89/32", Enabled: false},
	}
	internalIPRanges := []string{"10.0.0.0/16"}

	addSourceRangeWhitelistToAnnotations(annotations, allowedIPRanges, internalIPRanges)

	// Check if the function exits early and does not modify annotations
	if len(annotations.WhitelistSourceRange) != 0 || annotations.ConfigurationSnippet != "" {
		t.Errorf("Function did not exit early when all rules are disabled")
	}
}
