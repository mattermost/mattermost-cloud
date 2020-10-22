// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package store

import (
	"database/sql"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
)

const (
	clusterAnnotationTable      = "ClusterAnnotation"
	installationAnnotationTable = "InstallationAnnotation"
)

var annotationSelect sq.SelectBuilder
var annotationColumns = []string{
	"Annotation.ID", "Annotation.Name",
}

var (
	// ErrClusterAnnotationUsedByInstallation is an error returned when user attempts to delete cluster annotation
	// present on the installation scheduled on that cluster.
	ErrClusterAnnotationUsedByInstallation = errors.New("cannot delete cluster annotation, " +
		"it is used by one or more installations scheduled on the cluster")
	// ErrInstallationAnnotationDoNotMatchClusters is an error returned when user attempts to add annotation to the
	// installation, that is not present on any of the clusters on which the installation is scheduled.
	ErrInstallationAnnotationDoNotMatchClusters = errors.New("cannot add annotations to installation, " +
		"one or more clusters on which installation is scheduled do not contain one or more of new annotations")
)

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

// GetOrCreateAnnotations fetches annotations by name or creates them if they do not exist.
func (sqlStore *SQLStore) GetOrCreateAnnotations(annotations []*model.Annotation) ([]*model.Annotation, error) {
	return sqlStore.getOrCreateAnnotations(sqlStore.db, annotations)
}

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
	annotations, err := sqlStore.getOrCreateAnnotations(sqlStore.db, annotations)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get or create annotations")
	}

	return sqlStore.createClusterAnnotations(sqlStore.db, clusterID, annotations)
}

func (sqlStore *SQLStore) createClusterAnnotations(db execer, clusterID string, annotations []*model.Annotation) ([]*model.Annotation, error) {
	builder := sq.Insert(clusterAnnotationTable).
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
	return sqlStore.getAnnotationsForCluster(sqlStore.db, clusterID)
}

func (sqlStore *SQLStore) getAnnotationsForCluster(db dbInterface, clusterID string) ([]*model.Annotation, error) {
	var annotations []*model.Annotation

	builder := sq.Select(annotationColumns...).
		From(clusterAnnotationTable).
		Where("ClusterID = ?", clusterID).
		LeftJoin("Annotation ON Annotation.ID=AnnotationID")
	err := sqlStore.selectBuilder(db, &annotations, builder)
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
		LeftJoin(fmt.Sprintf("%s ON %s.ClusterID = Cluster.ID", clusterAnnotationTable, clusterAnnotationTable)).
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

// DeleteClusterAnnotation removes annotation from a given cluster.
// Annotation cannot be removed if it is present on any of the Installations scheduled on the cluster.
func (sqlStore *SQLStore) DeleteClusterAnnotation(clusterID string, annotationName string) error {
	annotation, err := sqlStore.GetAnnotationByName(annotationName)
	if err != nil {
		return errors.Wrapf(err, "failed to get annotation '%s' by name", annotationName)
	}
	if annotation == nil {
		return nil
	}

	tx, err := sqlStore.beginCustomTransaction(sqlStore.db, &sql.TxOptions{Isolation: sql.LevelRepeatableRead})
	if err != nil {
		return errors.Wrap(err, "failed to begin the transaction")
	}
	defer tx.RollbackUnlessCommitted()

	clusterInstallations, err := sqlStore.getClusterInstallations(tx, &model.ClusterInstallationFilter{
		PerPage:        model.AllPerPage,
		ClusterID:      clusterID,
		IncludeDeleted: false},
	)
	if err != nil {
		return errors.Wrap(err, "failed to get cluster installations")
	}

	for _, ci := range clusterInstallations {
		annotations, err := sqlStore.getAnnotationsForInstallation(tx, ci.InstallationID)
		if err != nil {
			return errors.Wrapf(err, "failed to get annotations for '%s' installation", ci.InstallationID)
		}
		if model.ContainsAnnotation(annotations, annotation) {
			return ErrClusterAnnotationUsedByInstallation
		}
	}

	builder := sq.Delete(clusterAnnotationTable).
		Where("ClusterID = ?", clusterID).
		Where("AnnotationID = ?", annotation.ID)

	result, err := sqlStore.execBuilder(tx, builder)
	if err != nil {
		return errors.Wrap(err, "failed to delete cluster annotation")
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return errors.Wrap(err, "failed to check affected rows when deleting cluster annotation")
	}
	if rows > 1 { // Do not fail if annotation is not set on cluster
		return fmt.Errorf("error deleting cluster annotation, expected 0 or 1 rows to be affected was %d", rows)
	}

	err = tx.Commit()
	if err != nil {
		return errors.Wrap(err, "failed to commit the transaction")
	}

	return nil
}

// GetAnnotationsForInstallation fetches all annotations assigned to the installation.
func (sqlStore *SQLStore) GetAnnotationsForInstallation(installationID string) ([]*model.Annotation, error) {
	return sqlStore.getAnnotationsForInstallation(sqlStore.db, installationID)
}

func (sqlStore *SQLStore) getAnnotationsForInstallation(db dbInterface, installationID string) ([]*model.Annotation, error) {
	var annotations []*model.Annotation

	builder := sq.Select(annotationColumns...).
		From(installationAnnotationTable).
		Where("InstallationID = ?", installationID).
		LeftJoin("Annotation ON Annotation.ID=AnnotationID")
	err := sqlStore.selectBuilder(db, &annotations, builder)
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
		LeftJoin(fmt.Sprintf("%s ON %s.InstallationID = Installation.ID", installationAnnotationTable, installationAnnotationTable)).
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
// Annotation cannot be added to the Installation if any of the clusters on which the Installation is scheduled
// is not annotated with it.
func (sqlStore *SQLStore) CreateInstallationAnnotations(installationID string, annotations []*model.Annotation) ([]*model.Annotation, error) {
	tx, err := sqlStore.beginCustomTransaction(sqlStore.db, &sql.TxOptions{Isolation: sql.LevelRepeatableRead})
	if err != nil {
		return nil, errors.Wrap(err, "failed to begin the transaction")
	}
	defer tx.RollbackUnlessCommitted()

	annotations, err = sqlStore.getOrCreateAnnotations(tx, annotations)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get or create annotations")
	}

	clusterInstallations, err := sqlStore.getClusterInstallations(tx, &model.ClusterInstallationFilter{
		PerPage:        model.AllPerPage,
		InstallationID: installationID,
		IncludeDeleted: false},
	)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get cluster installations")
	}

	for _, ci := range clusterInstallations {
		clusterAnnotations, err := sqlStore.getAnnotationsForCluster(tx, ci.ClusterID)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get annotations for '%s' cluster", ci.ClusterID)
		}

		if !containsAllAnnotations(clusterAnnotations, annotations) {
			return nil, ErrInstallationAnnotationDoNotMatchClusters
		}
	}

	annotations, err = sqlStore.createInstallationAnnotations(tx, installationID, annotations)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create installation annotations")
	}

	err = tx.Commit()
	if err != nil {
		return nil, errors.Wrap(err, "failed to commit the transaction")
	}

	return annotations, nil
}

func (sqlStore *SQLStore) createInstallationAnnotations(db execer, installationID string, annotations []*model.Annotation) ([]*model.Annotation, error) {
	builder := sq.Insert(installationAnnotationTable).
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

// DeleteInstallationAnnotation removes annotation from a given Installation.
func (sqlStore *SQLStore) DeleteInstallationAnnotation(installationID string, annotationName string) error {
	annotation, err := sqlStore.GetAnnotationByName(annotationName)
	if err != nil {
		return errors.Wrapf(err, "failed to get annotation '%s' by name", annotationName)
	}
	if annotation == nil {
		return nil
	}

	builder := sq.Delete(installationAnnotationTable).
		Where("InstallationID = ?", installationID).
		Where("AnnotationID = ?", annotation.ID)

	result, err := sqlStore.execBuilder(sqlStore.db, builder)
	if err != nil {
		return errors.Wrap(err, "failed to delete installation annotation")
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return errors.Wrap(err, "failed to check affected rows when deleting installation annotation")
	}
	if rows > 1 { // Do not fail if annotation is not set on installation
		return fmt.Errorf("error deleting installation annotation, expected 0 or 1 rows to be affected was %d", rows)
	}

	return nil
}

func containsAllAnnotations(base, new []*model.Annotation) bool {
	for _, n := range new {
		if !model.ContainsAnnotation(base, n) {
			return false
		}
	}
	return true
}
