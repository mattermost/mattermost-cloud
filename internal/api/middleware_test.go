package api

import (
	"testing"
)

func TestIsAccessAllowed(t *testing.T) {
	restrictedEndpoints := []string{
		"api/installation",
		"/api/installation",
		"api/cluster_installation",
		"/api/cluster_installation",
		"api/webhooks",
		"/api/webhooks",
		"/api/webhook",
		"api/webhook",
		"^/api/security/installation/[a-zA-Z0-9]{26}/deletion/lock$",
		"^/api/security/installation/[a-zA-Z0-9]{26}/deletion/unlock$",
	}

	restrictedClientIDs := []string{"restricted-client-id"}

	testCases := []struct {
		name          string
		clientID      string
		endpoint      string
		expectedAllow bool
	}{
		// Restricted client accesses wildcard prefix matches (should be allowed)
		{"Restricted client, wildcard prefix match 1 (allowed)", "restricted-client-id", "/api/installation/123", true},
		{"Restricted client, wildcard prefix match 2 (allowed)", "restricted-client-id", "api/cluster_installation/abc", true},
		{"Restricted client, wildcard prefix match 3 (allowed)", "restricted-client-id", "/api/webhooks/some-webhook", true},
		{"Restricted client, wildcard prefix match 4 (allowed)", "restricted-client-id", "api/webhook/another-webhook", true},

		// Exact regex matches (should be allowed)
		{"Restricted client, exact regex match 1 (allowed)", "restricted-client-id", "/api/security/installation/abcdefghijklmnopqrstuvwxyz/deletion/lock", true},
		{"Restricted client, exact regex match 2 (allowed)", "restricted-client-id", "/api/security/installation/zyxwvutsrqponmlkjihgfedcba/deletion/unlock", true},

		// Non-restricted client (should be allowed)
		{"Non-restricted client, wildcard prefix", "non-restricted-client", "/api/installation/123", true},
		{"Non-restricted client, exact regex", "non-restricted-client", "/api/security/installation/abcdefghijklmnopqrstuvwxyz/deletion/lock", true},

		// Restricted client, accessing endpoint not in allow-list (should be denied)
		{"Restricted client, other endpoint 1", "restricted-client-id", "/api/allowed/endpoint", false},
		{"Restricted client, other endpoint 2", "restricted-client-id", "/api/another/allowed/path", false},

		// Restricted client, accessing close to but not exact match endpoints from allow-list (should be denied)
		{"Restricted client, denied access 1", "restricted-client-id", "/api/security/installation/abcdefghijklmnopqrstuvwxyz/deletion", false},
		{"Restricted client, denied access 2", "restricted-client-id", "/api/security/installation/1234567890/some/other/action", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			allowed := isAccessAllowed(tc.clientID, tc.endpoint, restrictedClientIDs, restrictedEndpoints)
			if allowed != tc.expectedAllow {
				t.Errorf("Expected isAccessAllowed(%q, %q, %v, %v) to be %v, got %v", tc.clientID, tc.endpoint, restrictedClientIDs, restrictedEndpoints, tc.expectedAllow, allowed)
			}
		})
	}
}
