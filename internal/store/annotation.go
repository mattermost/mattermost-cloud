// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package store

import (
	"database/sql"
	sq "github.com/Masterminds/squirrel"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
)

var annotationSelect sq.SelectBuilder
var annotationColumns = []string{
	"Annotation.ID", "Annotation.Name",
}

func init() {
	annotationSelect = sq.Select(annotationColumns...).
		From("Annotation")
}

// GetAnnotationByName fetches the given annotation by name.
func (sqlStore *SQLStore) GetAnnotationByName(name string) (*model.Annotation, error) {
	return sqlStore.getAnnotationByName(sqlStore.db, name)
}

func (sqlStore *SQLStore) getAnnotationByName(db queryer, name string) (*model.Annotation, error) {
	var annotation model.Annotation

	builder := annotationSelect.
		Where("Name = ?", name).
		Limit(1)
	err := sqlStore.getBuilder(db, &annotation, builder)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, errors.Wrap(err, "failed to get annotation by name")
	}

	return &annotation, nil
}

// CreateAnnotation records the given annotation to the database, assigning it a unique ID.
func (sqlStore *SQLStore) CreateAnnotation(annotation *model.Annotation) error {
	return sqlStore.createAnnotation(sqlStore.db, annotation)
}

func (sqlStore *SQLStore) createAnnotation(db execer, annotation *model.Annotation) error {
	annotation.ID = model.NewID()

	_, err := sqlStore.execBuilder(db, sq.Insert("Annotation").
		SetMap(map[string]interface{}{
			"ID":   annotation.ID,
			"Name": annotation.Name,
		}))
	if err != nil {
		return errors.Wrap(err, "failed to create annotation")
	}

	return nil
}

// getOrCreateAnnotations fetches annotations by name or creates them if they do not exist.
func (sqlStore *SQLStore) getOrCreateAnnotations(db dbInterface, annotations []*model.Annotation) ([]*model.Annotation, error) {
	for i, ann := range annotations {
		annotation, err := sqlStore.getOrCreateAnnotation(db, ann)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get or create annotation '%s'", ann.Name)
		}
		annotations[i] = annotation
	}

	return annotations, nil
}

func (sqlStore *SQLStore) getOrCreateAnnotation(db dbInterface, annotation *model.Annotation) (*model.Annotation, error) {
	fetched, err := sqlStore.getAnnotationByName(db, annotation.Name)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get annotation by name")
	}
	if fetched != nil {
		return fetched, nil
	}

	err = sqlStore.createAnnotation(db, annotation)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create annotation")
	}

	return annotation, nil
}

// CreateClusterAnnotations maps selected annotations to cluster and stores it in the database.
func (sqlStore *SQLStore) CreateClusterAnnotations(clusterID string, annotations []*model.Annotation) ([]*model.Annotation, error) {
	return sqlStore.createClusterAnnotations(sqlStore.db, clusterID, annotations)
}

func (sqlStore *SQLStore) createClusterAnnotations(db execer, clusterID string, annotations []*model.Annotation) ([]*model.Annotation, error) {
	builder := sq.Insert("ClusterAnnotation").
		Columns("ID", "ClusterID", "AnnotationID")

	for _, a := range annotations {
		builder = builder.Values(model.NewID(), clusterID, a.ID)
	}
	_, err := sqlStore.execBuilder(db, builder)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create cluster annotations")
	}

	return annotations, nil
}

// GetAnnotationsForCluster fetches all annotations assigned to the cluster.
func (sqlStore *SQLStore) GetAnnotationsForCluster(clusterID string) ([]*model.Annotation, error) {
	var annotations []*model.Annotation

	builder := sq.Select(annotationColumns...).
		From("ClusterAnnotation").
		Where("ClusterID = ?", clusterID).
		LeftJoin("Annotation ON Annotation.ID=AnnotationID")
	err := sqlStore.selectBuilder(sqlStore.db, &annotations, builder)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get annotations for Cluster")
	}

	return annotations, nil
}

type clusterAnnotation struct {
	ClusterID      string
	AnnotationID   string
	AnnotationName string
}

// GetAnnotationsForClusters fetches all annotations assigned to the clusters.
func (sqlStore *SQLStore) GetAnnotationsForClusters(filter *model.ClusterFilter) (map[string][]*model.Annotation, error) {
	var clusterAnnotations []*clusterAnnotation

	builder := sq.Select(
		"Cluster.ID as ClusterID",
		"Annotation.ID as AnnotationID",
		"Annotation.Name as AnnotationName").
		From("Cluster").
		LeftJoin("ClusterAnnotation ON ClusterAnnotation.ClusterID = Cluster.ID").
		Join("Annotation ON Annotation.ID=AnnotationID")
	builder = sqlStore.applyClustersFilter(builder, filter)

	err := sqlStore.selectBuilder(sqlStore.db, &clusterAnnotations, builder)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get annotations for Cluster")
	}

	annotations := map[string][]*model.Annotation{}
	for _, ca := range clusterAnnotations {
		annotations[ca.ClusterID] = append(
			annotations[ca.ClusterID],
			&model.Annotation{ID: ca.AnnotationID, Name: ca.AnnotationName},
		)
	}

	return annotations, nil
}

// GetAnnotationsForInstallation fetches all annotations assigned to the installation.
func (sqlStore *SQLStore) GetAnnotationsForInstallation(installationID string) ([]*model.Annotation, error) {
	var annotations []*model.Annotation

	builder := sq.Select(annotationColumns...).
		From("InstallationAnnotation").
		Where("InstallationID = ?", installationID).
		LeftJoin("Annotation ON Annotation.ID=AnnotationID")
	err := sqlStore.selectBuilder(sqlStore.db, &annotations, builder)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get annotations for Installation")
	}

	return annotations, nil
}

type installationAnnotation struct {
	InstallationID string
	AnnotationID   string
	AnnotationName string
}

// GetAnnotationsForInstallations fetches all annotations assigned to installations.
func (sqlStore *SQLStore) GetAnnotationsForInstallations(filter *model.InstallationFilter) (map[string][]*model.Annotation, error) {
	var installationAnnotations []*installationAnnotation

	builder := sq.Select(
		"Installation.ID as InstallationID",
		"Annotation.ID as AnnotationID",
		"Annotation.Name as AnnotationName").
		From("Installation").
		LeftJoin("InstallationAnnotation ON InstallationAnnotation.InstallationID = Installation.ID").
		Join("Annotation ON Annotation.ID=AnnotationID")
	builder = sqlStore.applyInstallationFilter(builder, filter)

	err := sqlStore.selectBuilder(sqlStore.db, &installationAnnotations, builder)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get annotations for Installation")
	}

	annotations := map[string][]*model.Annotation{}
	for _, ca := range installationAnnotations {
		annotations[ca.InstallationID] = append(
			annotations[ca.InstallationID],
			&model.Annotation{ID: ca.AnnotationID, Name: ca.AnnotationName},
		)
	}

	return annotations, nil
}

// CreateInstallationAnnotations maps selected annotations to installation and stores it in the database.
func (sqlStore *SQLStore) CreateInstallationAnnotations(installationID string, annotations []*model.Annotation) ([]*model.Annotation, error) {
	return sqlStore.createInstallationAnnotations(sqlStore.db, installationID, annotations)
}

func (sqlStore *SQLStore) createInstallationAnnotations(db execer, installationID string, annotations []*model.Annotation) ([]*model.Annotation, error) {
	builder := sq.Insert("InstallationAnnotation").
		Columns("ID", "InstallationID", "AnnotationID")

	for _, a := range annotations {
		builder = builder.Values(model.NewID(), installationID, a.ID)
	}
	_, err := sqlStore.execBuilder(db, builder)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create installation annotations")
	}

	return annotations, nil
}
