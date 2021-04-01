// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package provisioner

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/client-go/kubernetes/typed/batch/v1"
)

const (
	// Run job with only one attempt to avoid possibility of waking up workspace before retry.
	backupRestoreBackoffLimit int32 = 0
	backupAction                    = "backup"
	restoreAction                   = "restore"
)

// ErrJobBackoffLimitReached indicates that job failed all possible attempts and there is no reason for retrying.
var ErrJobBackoffLimitReached = errors.New("job reached backoff limit")

// BackupOperator provides methods to run, check and cleanup backup jobs.
type BackupOperator struct {
	jobTTLSecondsAfterFinish *int32
	backupRestoreImage       string
	awsRegion                string
}

// NewBackupOperator creates new BackupOperator.
func NewBackupOperator(image, region string, jobTTLSeconds int32) *BackupOperator {
	jobTTL := &jobTTLSeconds
	if jobTTLSeconds < 0 {
		jobTTL = nil
	}

	return &BackupOperator{
		jobTTLSecondsAfterFinish: jobTTL,
		backupRestoreImage:       image,
		awsRegion:                region,
	}
}

// TriggerBackup creates new backup job and waits for it to start.
func (o BackupOperator) TriggerBackup(
	jobsClient v1.JobInterface,
	backup *model.InstallationBackup,
	installation *model.Installation,
	fileStoreCfg *model.FilestoreConfig,
	dbSecret string,
	logger log.FieldLogger) (*model.S3DataResidence, error) {

	dataResidence := model.S3DataResidence{
		Region:    o.awsRegion,
		Bucket:    fileStoreCfg.Bucket,
		URL:       fileStoreCfg.URL,
		ObjectKey: backupObjectKey(backup.ID),
	}
	storageEndpoint := fileStoreCfg.URL

	var envVars []corev1.EnvVar

	if installation.Filestore == model.InstallationFilestoreBifrost {
		storageEndpoint = bifrostEndpoint
		envVars = bifrostEnvs()
	}
	if installation.Filestore == model.InstallationFilestoreBifrost ||
		installation.Filestore == model.InstallationFilestoreMultiTenantAwsS3 {
		dataResidence.PathPrefix = installation.ID
	}

	envVars = append(envVars, prepareEnvs(dataResidence, storageEndpoint, fileStoreCfg.Secret, dbSecret)...)

	backupJobName := makeJobName(backupAction, backup.ID)
	job := o.createBackupRestoreJob(backupJobName, installation.ID, backupAction, envVars)

	err := o.startJob(jobsClient, job, logger)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to start backup job")
	}

	return &dataResidence, nil
}

// CheckBackupStatus checks status of backup job,
// returns job start time, when the job finished or -1 if it is still running.
func (o BackupOperator) CheckBackupStatus(jobsClient v1.JobInterface, backup *model.InstallationBackup, logger log.FieldLogger) (int64, error) {
	return o.checkJobStatus(jobsClient, makeJobName(backupAction, backup.ID), logger, extractStartTime)
}

// CleanupBackupJob removes backup job from the cluster if it exists.
func (o BackupOperator) CleanupBackupJob(jobsClient v1.JobInterface, backup *model.InstallationBackup, logger log.FieldLogger) error {
	return o.cleanupJob(jobsClient, makeJobName(backupAction, backup.ID), logger)
}

// TriggerRestore creates new restore job and waits for it to start.
func (o BackupOperator) TriggerRestore(
	jobsClient v1.JobInterface,
	backup *model.InstallationBackup,
	installation *model.Installation,
	fileStoreCfg *model.FilestoreConfig,
	dbSecret string,
	logger log.FieldLogger) error {

	if backup.DataResidence == nil {
		return errors.New("Installation backup is invalid - data residence is nil")
	}

	storageEndpoint := backup.DataResidence.URL
	var envVars []corev1.EnvVar

	if installation.Filestore == model.InstallationFilestoreBifrost {
		storageEndpoint = bifrostEndpoint
		envVars = bifrostEnvs()
	}

	envVars = append(envVars, prepareEnvs(*backup.DataResidence, storageEndpoint, fileStoreCfg.Secret, dbSecret)...)

	restoreJobName := makeJobName(restoreAction, backup.ID)
	job := o.createBackupRestoreJob(restoreJobName, installation.ID, restoreAction, envVars)

	err := o.startJob(jobsClient, job, logger)
	if err != nil {
		return errors.Wrap(err, "Failed to start restore job")
	}
	return nil
}

// CheckRestoreStatus checks status of restore job,
// returns job start time, when the job finished or -1 if it is still running.
func (o BackupOperator) CheckRestoreStatus(jobsClient v1.JobInterface, backup *model.InstallationBackup, logger log.FieldLogger) (int64, error) {
	return o.checkJobStatus(jobsClient, makeJobName(restoreAction, backup.ID), logger, extractCompletionTime)
}

// CleanupRestoreJob removes restore job from the cluster if it exists.
func (o BackupOperator) CleanupRestoreJob(jobsClient v1.JobInterface, backup *model.InstallationBackup, logger log.FieldLogger) error {
	return o.cleanupJob(jobsClient, makeJobName(restoreAction, backup.ID), logger)
}

// startJob creates job if does not exists and waits for it to start.
func (o BackupOperator) startJob(jobsClient v1.JobInterface, job *batchv1.Job, logger log.FieldLogger) error {
	jobName := job.Name

	ctx := context.Background()
	job, err := jobsClient.Create(ctx, job, metav1.CreateOptions{})
	if err != nil {
		if !k8sErrors.IsAlreadyExists(err) {
			return errors.Wrap(err, "failed to create restore job")
		}
		logger.Warnf("Job %q already exists", jobName)
	}

	// Wait for 5 seconds for job to start, if it won't it will be caught on next round.
	err = wait.Poll(time.Second, 5*time.Second, func() (bool, error) {
		job, err = jobsClient.Get(ctx, jobName, metav1.GetOptions{})
		if err != nil {
			return false, errors.Wrapf(err, "failed to get %q job", jobName)
		}
		if job.Status.Active == 0 && job.Status.CompletionTime == nil {
			logger.Infof("Job %q not yet started", jobName)
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		return errors.Wrapf(err, "Job %q not yet started", jobName)
	}
	return nil
}

func (o BackupOperator) checkJobStatus(
	jobsClient v1.JobInterface,
	jobName string,
	logger log.FieldLogger,
	extractTimestampFunc func(job *batchv1.Job, logger log.FieldLogger) int64) (int64, error) {
	ctx := context.Background()
	job, err := jobsClient.Get(ctx, jobName, metav1.GetOptions{})
	if err != nil {
		return -1, errors.Wrap(err, "failed to get job")
	}

	if job.Status.Succeeded > 0 {
		logger.Info("Job finished with success")
		return extractTimestampFunc(job, logger), nil
	}

	if job.Status.Failed > 0 {
		logger.Warnf("Job failed %d times", job.Status.Failed)
	}

	if job.Status.Active > 0 {
		logger.Info("Job is still running")
		return -1, nil
	}

	if job.Status.Failed == 0 {
		logger.Info("Job not started yet")
		return -1, nil
	}

	backoffLimit := getInt32(job.Spec.BackoffLimit)
	if job.Status.Failed > backoffLimit {
		logger.Error("Job reached backoff limit")
		return -1, ErrJobBackoffLimitReached
	}

	logger.Infof("Job waiting for retry, will be retried at most %d more times", backoffLimit+1-job.Status.Failed)
	return -1, nil
}

func (o BackupOperator) cleanupJob(jobsClient v1.JobInterface, jobName string, logger log.FieldLogger) error {
	deletePropagation := metav1.DeletePropagationBackground
	ctx := context.Background()
	err := jobsClient.Delete(ctx, jobName, metav1.DeleteOptions{PropagationPolicy: &deletePropagation})
	if k8sErrors.IsNotFound(err) {
		logger.Warnf("Job %q does not exist, assuming already deleted", jobName)
		return nil
	}
	if err != nil {
		return errors.Wrap(err, "failed to delete job")
	}

	logger.Info("Job successfully marked for deletion")
	return nil
}

func (o BackupOperator) createBackupRestoreJob(name, namespace, action string, envs []corev1.EnvVar) *batchv1.Job {
	backoff := backupRestoreBackoffLimit

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    map[string]string{"app": "backup-restore"},
		},
		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"app": "backup-restore"},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						o.createBackupRestoreContainer(action, envs),
					},
					RestartPolicy: corev1.RestartPolicyNever,
				},
			},
			BackoffLimit:            &backoff,
			TTLSecondsAfterFinished: o.jobTTLSecondsAfterFinish,
		},
	}

	return job
}

func (o BackupOperator) createBackupRestoreContainer(action string, envs []corev1.EnvVar) corev1.Container {
	return corev1.Container{
		Name:  "backup-restore",
		Image: o.backupRestoreImage,
		Args:  []string{action},
		Env:   envs,
		Resources: corev1.ResourceRequirements{
			Limits: corev1.ResourceList{
				corev1.ResourceEphemeralStorage: resource.MustParse("15Gi"), // This should be enough even for very large installations
			},
			Requests: corev1.ResourceList{
				corev1.ResourceEphemeralStorage: resource.MustParse("2Gi"),
			},
		},
	}
}

func prepareEnvs(dataRes model.S3DataResidence, endpoint string, fileStoreSecret, dbSecret string) []corev1.EnvVar {
	envs := []corev1.EnvVar{
		{
			Name:  "BRT_STORAGE_REGION",
			Value: dataRes.Region,
		},
		{
			Name:  "BRT_STORAGE_BUCKET",
			Value: dataRes.Bucket,
		},
		{
			Name:  "BRT_STORAGE_ENDPOINT",
			Value: endpoint,
		},
		{
			Name:  "BRT_STORAGE_PATH_PREFIX",
			Value: dataRes.PathPrefix,
		},
		{
			Name:  "BRT_STORAGE_OBJECT_KEY",
			Value: dataRes.ObjectKey,
		},
		{
			Name:      "BRT_DATABASE",
			ValueFrom: envSourceFromSecret(dbSecret, "DB_CONNECTION_STRING"),
		},
		{
			Name:      "BRT_STORAGE_ACCESS_KEY",
			ValueFrom: envSourceFromSecret(fileStoreSecret, "accesskey"),
		},
		{
			Name:      "BRT_STORAGE_SECRET_KEY",
			ValueFrom: envSourceFromSecret(fileStoreSecret, "secretkey"),
		},
	}

	return envs
}

func bifrostEnvs() []corev1.EnvVar {
	return []corev1.EnvVar{
		{
			Name:  "BRT_STORAGE_TLS",
			Value: strconv.FormatBool(false),
		},
		{
			Name:  "BRT_STORAGE_TYPE",
			Value: "bifrost",
		},
	}
}

func envSourceFromSecret(secretName, key string) *corev1.EnvVarSource {
	return &corev1.EnvVarSource{
		SecretKeyRef: &corev1.SecretKeySelector{
			LocalObjectReference: corev1.LocalObjectReference{
				Name: secretName,
			},
			Key: key,
		},
	}
}

func extractStartTime(job *batchv1.Job, logger log.FieldLogger) int64 {
	if job.Status.StartTime != nil {
		return asMillis(*job.Status.StartTime)
	}

	logger.Warn("failed to get job start time, using creation timestamp")
	return asMillis(job.CreationTimestamp)
}

func extractCompletionTime(job *batchv1.Job, logger log.FieldLogger) int64 {
	if job.Status.CompletionTime != nil {
		return asMillis(*job.Status.CompletionTime)
	}

	logger.Warn("failed to get job completion time, using current time")
	return asMillis(metav1.Now())
}

func backupObjectKey(id string) string {
	return fmt.Sprintf("backup-%s", id)
}

func makeJobName(action, id string) string {
	return fmt.Sprintf("database-%s-%s", action, id)
}

func getInt32(i32 *int32) int32 {
	if i32 == nil {
		return 0
	}
	return *i32
}

// asMillis returns metav1.Time as milliseconds.
func asMillis(t metav1.Time) int64 {
	return t.UnixNano() / int64(time.Millisecond)
}
