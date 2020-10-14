// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package model

import (
	"fmt"
	"regexp"
	"sort"
)

const (
	annotationMinLen        = 3
	annotationMaxLen        = 64
	annotationAllowedFormat = "annotations must start with letter and can contain only lowercase letters, numbers or '_', '-' characters"
)

var annotationRegex = regexp.MustCompile("^[a-z].[a-z0-9_-]*$")

// Annotation represents an annotation.
type Annotation struct {
	ID   string
	Name string
}

// AnnotationsFromStringSlice converts list of strings to list of annotations
func AnnotationsFromStringSlice(names []string) ([]*Annotation, error) {
	if names == nil {
		return nil, nil
	}

	annotations := make([]*Annotation, 0, len(names))
	for _, n := range names {
		if len(n) < annotationMinLen || len(n) > annotationMaxLen {
			return nil, fmt.Errorf("error: annotation '%s' is invalid: annotations must be between %d and %d characters long", n, annotationMinLen, annotationMaxLen)
		}
		if !annotationRegex.MatchString(n) {
			return nil, fmt.Errorf("error: annotation '%s' is invalid: %s", n, annotationAllowedFormat)
		}
		annotations = append(annotations, &Annotation{Name: n})
	}

	return annotations, nil
}

// SortAnnotations sorts annotations by name alphabetically.
func SortAnnotations(annotations []*Annotation) []*Annotation {
	sort.Slice(annotations, func(i, j int) bool {
		return annotations[i].Name < annotations[j].Name
	})
	return annotations
}
