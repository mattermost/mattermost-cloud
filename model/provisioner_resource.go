// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package model

// Resource is a generic interface implemented by all resources
// supervised by Provisioner.
type Resource interface {
	GetID() string
	IsDeleted() bool
}

// SupervisedResource is a resource with dedicated supervisor.
type SupervisedResource interface {
	Resource
	GetState() string
}

// Resources is a collection of Resource objects.
type Resources []Resource

// GetIDs returns IDs of all resources in the collection.
func GetIDs(resources []Resource) []string {
	ids := make([]string, 0, len(resources))
	for _, elem := range resources {
		ids = append(ids, elem.GetID())
	}
	return ids
}
