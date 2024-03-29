// Copyright (c) Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//
// Code generated by generator, DO NOT EDIT.

package model

// GetID returns ID of the resource.
func (c *Cluster) GetID() string {
	return c.ID
}

// GetState returns State of the resource.
func (c *Cluster) GetState() string {
	return string(c.State)
}

// IsDeleted determines whether the resource is deleted.
func (c *Cluster) IsDeleted() bool {
	return c.DeleteAt > 0
}

// ClustersAsResources returns collection as Resource objects.
func ClustersAsResources(collection []*Cluster) []Resource {
	resources := make([]Resource, 0, len(collection))
	for _, elem := range collection {
		resources = append(resources, Resource(elem))
	}
	return resources
}
