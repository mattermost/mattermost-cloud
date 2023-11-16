// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package provisioner

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/mattermost/mattermost-cloud/model"
)

//	func TestAddSourceRangeWhitelistToAnnotations(t *testing.T) {
//		t.Run("nil allowed ranges, blank internal ranges", func(t *testing.T) {
//			annotations := getIngressAnnotations()
//			addSourceRangeWhitelistToAnnotations(annotations, nil, []string{""})
//			require.Equal(t, getIngressAnnotations(), annotations)
//		})
//
//		t.Run("nil allowed ranges, internal ranges", func(t *testing.T) {
//			annotations := getIngressAnnotations()
//			addSourceRangeWhitelistToAnnotations(annotations, nil, []string{"2.2.2.2/24"})
//			require.Equal(t, getIngressAnnotations(), annotations)
//		})
//
//		t.Run("allowed ranges, blank internal ranges", func(t *testing.T) {
//			annotations := getIngressAnnotations()
//			allowedRanges := &model.AllowedIPRanges{{CIDRBlock: "1.1.1.1/24", Enabled: true}}
//			addSourceRangeWhitelistToAnnotations(annotations, allowedRanges, nil)
//			require.Equal(t, []string{"1.1.1.1/24"}, annotations.WhitelistSourceRange)
//			expectedAnnotations := getIngressAnnotations()
//			expectedAnnotations.WhitelistSourceRange = []string{"1.1.1.1/24"}
//			require.Equal(t, annotations, expectedAnnotations)
//		})
//
//		t.Run("allowed range, internal range", func(t *testing.T) {
//			annotations := getIngressAnnotations()
//			allowedRanges := &model.AllowedIPRanges{{CIDRBlock: "1.1.1.1/24", Enabled: true}}
//			addSourceRangeWhitelistToAnnotations(annotations, allowedRanges, []string{"2.2.2.2/24"})
//			require.Equal(t, []string{"1.1.1.1/24", "2.2.2.2/24"}, annotations.WhitelistSourceRange)
//			expectedAnnotations := getIngressAnnotations()
//			expectedAnnotations.WhitelistSourceRange = []string{"1.1.1.1/24", "2.2.2.2/24"}
//			require.Equal(t, annotations, expectedAnnotations)
//		})
//
//		t.Run("multiple of both ranges", func(t *testing.T) {
//			annotations := getIngressAnnotations()
//			allowedRanges := &model.AllowedIPRanges{
//				{CIDRBlock: "1.1.1.1/24", Enabled: true},
//				{CIDRBlock: "1.1.1.2/24", Enabled: true},
//			}
//			addSourceRangeWhitelistToAnnotations(annotations, allowedRanges, []string{"2.2.2.2/24", "2.2.2.3/24"})
//			require.Equal(t, []string{"1.1.1.1/24", "1.1.1.2/24", "2.2.2.2/24", "2.2.2.3/24"}, annotations.WhitelistSourceRange)
//			expectedAnnotations := getIngressAnnotations()
//			expectedAnnotations.WhitelistSourceRange = []string{"1.1.1.1/24", "1.1.1.2/24", "2.2.2.2/24", "2.2.2.3/24"}
//			require.Equal(t, annotations, expectedAnnotations)
//		})
//
//		t.Run("multiple of both ranges, some disabled allowed ranges", func(t *testing.T) {
//			annotations := getIngressAnnotations()
//			allowedRanges := &model.AllowedIPRanges{
//				{CIDRBlock: "1.1.1.1/24", Enabled: true},
//				{CIDRBlock: "1.1.1.2/24", Enabled: false},
//			}
//			addSourceRangeWhitelistToAnnotations(annotations, allowedRanges, []string{"2.2.2.2/24", "2.2.2.3/24"})
//			require.Equal(t, []string{"1.1.1.1/24", "2.2.2.2/24", "2.2.2.3/24"}, annotations.WhitelistSourceRange)
//			expectedAnnotations := getIngressAnnotations()
//			expectedAnnotations.WhitelistSourceRange = []string{"1.1.1.1/24", "2.2.2.2/24", "2.2.2.3/24"}
//			require.Equal(t, annotations, expectedAnnotations)
//		})
//	}
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

	// Check if ConfigurationSnippet contains the correct 'allow' directives and custom 419 logic
	for _, ip := range expectedIPs {
		if !strings.Contains(annotations.ConfigurationSnippet, fmt.Sprintf("allow %s;", ip)) {
			t.Errorf("ConfigurationSnippet missing allow directive for %s", ip)
		}
	}
	if !strings.Contains(annotations.ConfigurationSnippet, "return 419;") {
		t.Errorf("ConfigurationSnippet missing custom 419 logic")
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
