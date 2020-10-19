// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package store

import (
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
)

// GetClusterDTO fetches the given cluster by id with data from connected tables.
func (sqlStore *SQLStore) GetClusterDTO(id string) (*model.ClusterDTO, error) {
	cluster, err := sqlStore.GetCluster(id)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get cluster")
	}
	if cluster == nil {
		return nil, nil
	}

	annotation, err := sqlStore.GetAnnotationsForCluster(id)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get annotations for cluster")
	}

	return &model.ClusterDTO{
		Cluster:     cluster,
		Annotations: annotation,
	}, nil
}

// GetClusterDTOs fetches the given page of clusters with data from connected tables. The first page is 0.
func (sqlStore *SQLStore) GetClusterDTOs(filter *model.ClusterFilter) ([]*model.ClusterDTO, error) {
	clusters, err := sqlStore.GetClusters(filter)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get clusters")
	}

	annotations, err := sqlStore.GetAnnotationsForClusters(filter)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get annotations for clusters")
	}

	dtos := make([]*model.ClusterDTO, 0, len(clusters))
	for _, c := range clusters {
		dtos = append(dtos, &model.ClusterDTO{Cluster: c, Annotations: annotations[c.ID]})
	}

	return dtos, nil
}
