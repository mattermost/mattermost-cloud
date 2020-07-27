// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package model

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMultitenantDatabaseInstallationsCountContainsAndAdd(t *testing.T) {
	var installations MultitenantDatabaseInstallations
	for i := 1; i < 10; i++ {
		t.Run(fmt.Sprintf("count-%d", i), func(t *testing.T) {
			newID := NewID()
			installations.Add(newID)
			assert.Contains(t, installations, newID)
			assert.Equal(t, installations.Count(), i)
		})
	}
}

func TestMultitenantDatabaseInstallationsContains(t *testing.T) {
	standardInstallations := MultitenantDatabaseInstallations{
		"id1", "id2", "id3",
	}
	var testCases = []struct {
		id             string
		installations  MultitenantDatabaseInstallations
		expectContains bool
	}{
		{"", standardInstallations, false},
		{"id1", MultitenantDatabaseInstallations{}, false},
		{"id4", standardInstallations, false},
		{"id1", standardInstallations, true},
		{"id3", standardInstallations, true},
		{"id4", standardInstallations, false},
	}

	for _, tc := range testCases {
		t.Run(tc.id, func(t *testing.T) {
			assert.Equal(t, tc.expectContains, tc.installations.Contains(tc.id))
		})
	}
}

func TestMultitenantDatabaseInstallationsAdd(t *testing.T) {
	var testCases = []struct {
		id            string
		installations MultitenantDatabaseInstallations
	}{
		{"id1", MultitenantDatabaseInstallations{}},
		{"id4", MultitenantDatabaseInstallations{"id1", "id2", "id3"}},
		{"id5", MultitenantDatabaseInstallations{"id1", "id2", "id3"}},
	}

	for _, tc := range testCases {
		t.Run(tc.id, func(t *testing.T) {
			assert.NotContains(t, tc.installations, tc.id)
			tc.installations.Add(tc.id)
			assert.Contains(t, tc.installations, tc.id)
		})
	}
}

func TestMultitenantDatabaseInstallationsRemove(t *testing.T) {
	var testCases = []struct {
		id            string
		installations MultitenantDatabaseInstallations
	}{
		{"id1", MultitenantDatabaseInstallations{}},
		{"id1", MultitenantDatabaseInstallations{"id1", "id2", "id3"}},
		{"id2", MultitenantDatabaseInstallations{"id1", "id2", "id3"}},
		{"id3", MultitenantDatabaseInstallations{"id1", "id2", "id3"}},
		{"id4", MultitenantDatabaseInstallations{"id1", "id2", "id3"}},
	}

	for _, tc := range testCases {
		t.Run(tc.id, func(t *testing.T) {
			tc.installations.Remove(tc.id)
			assert.NotContains(t, tc.installations, tc.id)
		})
	}
}
