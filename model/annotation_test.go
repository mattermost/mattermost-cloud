// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package model

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAnnotationsFromStringSlice(t *testing.T) {

	t.Run("valid annotations", func(t *testing.T) {
		for _, testCase := range []struct {
			description string
			names       []string
			annotations []*Annotation
		}{
			{"nil array", nil, nil},
			{"empty array", []string{}, []*Annotation{}},
			{
				"valid names",
				[]string{"abcd", "multi-tenant", "awesome_annotation"},
				[]*Annotation{{Name: "abcd"}, {Name: "multi-tenant"}, {Name: "awesome_annotation"}},
			},
			{
				"long names",
				[]string{"multi-tenant-1234-abcd-very-long-name", "super-awesome-long_name"},
				[]*Annotation{{Name: "multi-tenant-1234-abcd-very-long-name"}, {Name: "super-awesome-long_name"}},
			},
		} {
			t.Run(testCase.description, func(t *testing.T) {
				annotations, err := AnnotationsFromStringSlice(testCase.names)
				require.NoError(t, err)
				assert.Equal(t, testCase.annotations, annotations)
			})
		}
	})

	t.Run("invalid annotations", func(t *testing.T) {
		for _, testCase := range []struct {
			description string
			names       []string
		}{
			{
				"to long name",
				[]string{"abcd", "my-annotation-1-with-super-long-name-that-is-not-allowed-but-cool"},
			},
			{
				"upper case letter",
				[]string{"abcd", "xyz", "Abcd"},
			},
			{"to short name", []string{"abcd", "ab"}},
			{"invalid characters", []string{"a?bcd"}},
			{"not allowed character ' '", []string{"ab cd"}},
			{"not allowed character '!'", []string{"ab!cd"}},
			{"not allowed character ':'", []string{"ab:cd"}},
			{"not allowed character '{'", []string{"ab{cd}"}},
			{"not allowed character '+'", []string{"ab+cd"}},
			{"starts with '_'", []string{"_abcd"}},
			{"starts with '-'", []string{"-abcd"}},
			{"starts with number", []string{"6abcd"}},
		} {
			t.Run(testCase.description, func(t *testing.T) {
				annotations, err := AnnotationsFromStringSlice(testCase.names)
				require.Error(t, err)
				assert.Nil(t, annotations)
			})
		}
	})

}

func TestSortAnnotations(t *testing.T) {

	for _, testCase := range []struct {
		description string
		annotations []*Annotation
		expected    []*Annotation
	}{
		{
			description: "sort annotations",
			annotations: []*Annotation{
				{Name: "xyz"}, {Name: "other-annotation"}, {Name: "other_annotation"}, {Name: "abcdefgh"}, {Name: "abcd"},
			},
			expected: []*Annotation{
				{Name: "abcd"}, {Name: "abcdefgh"}, {Name: "other-annotation"}, {Name: "other_annotation"}, {Name: "xyz"},
			},
		},
	} {
		t.Run(testCase.description, func(t *testing.T) {
			SortAnnotations(testCase.annotations)
			assert.Equal(t, testCase.expected, testCase.annotations)
		})
	}
}

func TestNewAddAnnotationsRequestFromReader(t *testing.T) {
	t.Run("empty request", func(t *testing.T) {
		annotationsRequest, err := NewAddAnnotationsRequestFromReader(bytes.NewReader([]byte(
			``,
		)))
		require.NoError(t, err)
		require.Equal(t, &AddAnnotationsRequest{}, annotationsRequest)
	})

	t.Run("invalid request", func(t *testing.T) {
		annotationsRequest, err := NewAddAnnotationsRequestFromReader(bytes.NewReader([]byte(
			`{test`,
		)))
		require.Error(t, err)
		require.Nil(t, annotationsRequest)
	})

	t.Run("request", func(t *testing.T) {
		annotationsRequest, err := NewAddAnnotationsRequestFromReader(bytes.NewReader([]byte(
			`{"annotations":["abcd", "super-awesome"]}`,
		)))
		require.NoError(t, err)
		require.Equal(t, &AddAnnotationsRequest{Annotations: []string{"abcd", "super-awesome"}}, annotationsRequest)
	})
}

func TestContainsAnnotation(t *testing.T) {

	annotations := []*Annotation{
		{ID: "1", Name: "annotation1"},
		{ID: "2", Name: "my-annotation"},
		{ID: "3", Name: "super-awesome"},
		{ID: "4", Name: "fourth"},
		{ID: "5", Name: "multi_tenant"},
	}

	for _, testCase := range []struct {
		description string
		slice       []*Annotation
		annotation  *Annotation
		isPresent   bool
	}{
		{
			description: "should find annotation",
			slice:       annotations,
			annotation:  &Annotation{ID: "3", Name: "super-awesome"},
			isPresent:   true,
		},
		{
			description: "should find with ID only",
			slice:       annotations,
			annotation:  &Annotation{ID: "5"},
			isPresent:   true,
		},
		{
			description: "should not find annotation",
			slice:       annotations,
			annotation:  &Annotation{ID: "10", Name: "fourth"},
			isPresent:   false,
		},
		{
			description: "should not find in empty slice",
			slice:       []*Annotation{},
			annotation:  &Annotation{ID: "1"},
			isPresent:   false,
		},
	} {
		t.Run(testCase.description, func(t *testing.T) {
			found := ContainsAnnotation(testCase.slice, testCase.annotation)
			assert.Equal(t, testCase.isPresent, found)
		})
	}
}
