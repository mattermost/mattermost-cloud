// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package model

import (
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"sort"

	"github.com/pkg/errors"
)

const (
	annotationMinLen        = 3
	annotationMaxLen        = 64
	annotationAllowedFormat = "annotations must start with a letter and can contain only lowercase letters, numbers or '_', '-' characters"
)

var annotationRegex = regexp.MustCompile("^[a-z]+[a-z0-9_-]*$")

// Annotation represents an annotation.
type Annotation struct {
	ID   string
	Name string
}

// AddAnnotationsRequest represent parameters passed to add set of annotations to the Cluster or Installation.
type AddAnnotationsRequest struct {
	Annotations []string `json:"annotations"`
}

// AnnotationsFromStringSlice converts list of strings to list of annotations.
func AnnotationsFromStringSlice(names []string) ([]*Annotation, error) {
	if names == nil {
		return nil, nil
	}

	annotations := make([]*Annotation, 0, len(names))
	for _, n := range names {
		err := validateAnnotationName(n)
		if err != nil {
			return nil, err
		}
		annotations = append(annotations, &Annotation{Name: n})
	}

	return annotations, nil
}

func validateAnnotationName(name string) error {
	if len(name) < annotationMinLen || len(name) > annotationMaxLen {
		return fmt.Errorf("annotation '%s' is invalid: annotations must be between %d and %d characters long", name, annotationMinLen, annotationMaxLen)
	}
	if !annotationRegex.MatchString(name) {
		return fmt.Errorf("annotation '%s' is invalid: %s", name, annotationAllowedFormat)
	}
	return nil
}

// SortAnnotations sorts annotations by name alphabetically.
func SortAnnotations(annotations []*Annotation) []*Annotation {
	sort.Slice(annotations, func(i, j int) bool {
		return annotations[i].Name < annotations[j].Name
	})
	return annotations
}

// GetAnnotationsIDs gets IDs of annotations.
func GetAnnotationsIDs(annotations []*Annotation) []string {
	ids := make([]string, 0, len(annotations))
	for _, ann := range annotations {
		ids = append(ids, ann.ID)
	}
	return ids
}

// NewAddAnnotationsRequestFromReader will create a AddAnnotationsRequest from an
// io.Reader with JSON data.
func NewAddAnnotationsRequestFromReader(reader io.Reader) (*AddAnnotationsRequest, error) {
	var addAnnotationsRequest AddAnnotationsRequest
	err := json.NewDecoder(reader).Decode(&addAnnotationsRequest)
	if err != nil && err != io.EOF {
		return nil, errors.Wrap(err, "failed to decode add annotations request")
	}

	return &addAnnotationsRequest, nil
}

// ContainsAnnotation determines whether slice of Annotations contains a specific annotation.
func ContainsAnnotation(annotations []*Annotation, annotation *Annotation) bool {
	for _, ann := range annotations {
		if ann.ID == annotation.ID {
			return true
		}
	}
	return false
}
