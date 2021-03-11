// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package provisioner

import (
	"context"
	"fmt"
	"testing"
	"time"

	k8sErrors "k8s.io/apimachinery/pkg/api/errors"

	"k8s.io/apimachinery/pkg/util/wait"
	v1 "k8s.io/client-go/kubernetes/typed/batch/v1"

	"github.com/mattermost/mattermost-cloud/model"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestOperator_TriggerBackup(t *testing.T) {
	fileStoreSecret := "file-store-secret"
	fileStoreCfg := &model.FilestoreConfig{
		URL:    "filestore.com",
		Bucket: "plastic",
		Secret: fileStoreSecret,
	}
	databaseSecret := "database-secret"

	backupMeta := &model.InstallationBackup{
		ID:             "backup-1",
		InstallationID: "installation-1",
		State:          model.InstallationBackupStateBackupRequested,
		RequestAt:      1,
	}

	operator := NewBackupOperator("mattermost/backup-restore:test", "us", 100)

	for _, testCase := range []struct {
		description          string
		installation         *model.Installation
		expectedFileStoreURL string
		extraEnvs            map[string]string
	}{
		{
			description:          "s3 installation",
			installation:         &model.Installation{ID: "installation-1", Filestore: model.InstallationFilestoreAwsS3},
			expectedFileStoreURL: "filestore.com",
		},
		{
			description:          "bifrost installation",
			installation:         &model.Installation{ID: "installation-1", Filestore: model.InstallationFilestoreBifrost},
			expectedFileStoreURL: "bifrost.bifrost:80",
			extraEnvs: map[string]string{
				"BRT_STORAGE_PATH_PREFIX": "installation-1",
				"BRT_STORAGE_TLS":         "false",
				"BRT_STORAGE_TYPE":        "bifrost",
			},
		},
	} {
		t.Run(testCase.description, func(t *testing.T) {
			k8sClient := fake.NewSimpleClientset()
			jobClinet := k8sClient.BatchV1().Jobs("installation-1")
			go setJobActiveWhenExists(t, jobClinet, "database-backup-backup-1")

			dataRes, err := operator.TriggerBackup(
				jobClinet,
				backupMeta,
				testCase.installation,
				fileStoreCfg,
				databaseSecret,
				logrus.New())
			require.NoError(t, err)

			assert.Equal(t, "filestore.com", dataRes.URL)
			assert.Equal(t, "us", dataRes.Region)
			assert.Equal(t, "plastic", dataRes.Bucket)
			assert.Equal(t, "backup-backup-1", dataRes.ObjectKey)

			createdJob, err := jobClinet.Get(context.Background(), "database-backup-backup-1", metav1.GetOptions{})
			require.NoError(t, err)

			assert.Equal(t, "backup-restore", createdJob.Labels["app"])
			assert.Equal(t, "installation-1", createdJob.Namespace)
			assert.Equal(t, backupRestoreBackoffLimit, *createdJob.Spec.BackoffLimit)
			assert.Equal(t, int32(100), *createdJob.Spec.TTLSecondsAfterFinished)

			podTemplate := createdJob.Spec.Template
			assert.Equal(t, "backup-restore", podTemplate.Labels["app"])
			assert.Equal(t, "mattermost/backup-restore:test", podTemplate.Spec.Containers[0].Image)

			envs := podTemplate.Spec.Containers[0].Env
			assertEnvVarEqual(t, "BRT_STORAGE_REGION", "us", envs)
			assertEnvVarEqual(t, "BRT_STORAGE_BUCKET", "plastic", envs)
			assertEnvVarEqual(t, "BRT_STORAGE_ENDPOINT", testCase.expectedFileStoreURL, envs)
			assertEnvVarEqual(t, "BRT_STORAGE_OBJECT_KEY", "backup-backup-1", envs)
			assertEnvVarFromSecret(t, "BRT_STORAGE_ACCESS_KEY", fileStoreSecret, "accesskey", envs)
			assertEnvVarFromSecret(t, "BRT_STORAGE_SECRET_KEY", fileStoreSecret, "secretkey", envs)
			assertEnvVarFromSecret(t, "BRT_DATABASE", databaseSecret, "DB_CONNECTION_STRING", envs)

			for k, v := range testCase.extraEnvs {
				assertEnvVarEqual(t, k, v, envs)
			}
		})
	}

	t.Run("succeed if job already exists", func(t *testing.T) {
		existing := &batchv1.Job{
			ObjectMeta: metav1.ObjectMeta{Name: "database-backup-backup-1", Namespace: "installation-1"},
			Status:     batchv1.JobStatus{Active: 1},
		}
		k8sClient := fake.NewSimpleClientset(existing)
		jobClinet := k8sClient.BatchV1().Jobs("installation-1")

		installation := &model.Installation{ID: "installation-1", Filestore: model.InstallationFilestoreMultiTenantAwsS3}

		dataRes, err := operator.TriggerBackup(
			jobClinet,
			backupMeta,
			installation,
			fileStoreCfg,
			databaseSecret,
			logrus.New())
		require.NoError(t, err)

		assert.Equal(t, "filestore.com", dataRes.URL)
		assert.Equal(t, "us", dataRes.Region)
		assert.Equal(t, "plastic", dataRes.Bucket)
		assert.Equal(t, "backup-backup-1", dataRes.ObjectKey)
	})

	t.Run("set ttl to nil if negative value", func(t *testing.T) {
		k8sClient := fake.NewSimpleClientset()
		jobClinet := k8sClient.BatchV1().Jobs("installation-1")
		go setJobActiveWhenExists(t, jobClinet, "database-backup-backup-1")

		installation := &model.Installation{ID: "installation-1", Filestore: model.InstallationFilestoreMultiTenantAwsS3}

		operator := NewBackupOperator("image", "us", -1)

		_, err := operator.TriggerBackup(
			jobClinet,
			backupMeta,
			installation,
			fileStoreCfg,
			databaseSecret,
			logrus.New())
		require.NoError(t, err)

		createdJob, err := jobClinet.Get(context.Background(), "database-backup-backup-1", metav1.GetOptions{})
		require.NoError(t, err)
		assert.Nil(t, createdJob.Spec.TTLSecondsAfterFinished)
	})

	t.Run("fail if job not ready", func(t *testing.T) {
		k8sClient := fake.NewSimpleClientset()
		jobClinet := k8sClient.BatchV1().Jobs("installation-1")
		installation := &model.Installation{ID: "installation-1", Filestore: model.InstallationFilestoreMultiTenantAwsS3}

		_, err := operator.TriggerBackup(
			jobClinet,
			backupMeta,
			installation,
			fileStoreCfg,
			databaseSecret,
			logrus.New())
		require.Error(t, err)
	})
}

func TestOperator_CheckBackupStatus(t *testing.T) {
	backupMeta := &model.InstallationBackup{
		ID:             "backup-1",
		InstallationID: "installation-1",
		State:          model.InstallationBackupStateBackupRequested,
		RequestAt:      1,
	}

	k8sClient := fake.NewSimpleClientset()
	jobClinet := k8sClient.BatchV1().Jobs("installation-1")

	operator := NewBackupOperator("mattermost/backup-restore:test", "us", 0)

	t.Run("error when job does not exists", func(t *testing.T) {
		_, err := operator.CheckBackupStatus(jobClinet, backupMeta, logrus.New())
		require.Error(t, err)
	})

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{Name: "database-backup-backup-1"},
		Spec:       batchv1.JobSpec{},
		Status:     batchv1.JobStatus{},
	}
	var err error
	job, err = jobClinet.Create(context.Background(), job, metav1.CreateOptions{})
	require.NoError(t, err)

	t.Run("return -1 start time if not finished", func(t *testing.T) {
		startTime, err := operator.CheckBackupStatus(jobClinet, backupMeta, logrus.New())
		require.NoError(t, err)
		assert.Equal(t, int64(-1), startTime)
	})

	job.Status.Failed = backupRestoreBackoffLimit + 1
	job, err = jobClinet.Update(context.Background(), job, metav1.UpdateOptions{})
	require.NoError(t, err)

	t.Run("ErrJobBackoffLimitReached when failed enough times", func(t *testing.T) {
		_, err = operator.CheckBackupStatus(jobClinet, backupMeta, logrus.New())
		require.Error(t, err)
		assert.Equal(t, ErrJobBackoffLimitReached, err)
	})

	expectedStartTime := metav1.Now()
	job.Status.Succeeded = 1
	job.Status.StartTime = &expectedStartTime
	job, err = jobClinet.Update(context.Background(), job, metav1.UpdateOptions{})
	require.NoError(t, err)

	t.Run("return start time when succeeded", func(t *testing.T) {
		startTime, err := operator.CheckBackupStatus(jobClinet, backupMeta, logrus.New())
		require.NoError(t, err)
		assert.Equal(t, asMillis(expectedStartTime), startTime)
	})
}

func TestBackupOperator_CleanupBackup(t *testing.T) {
	operator := NewBackupOperator("mattermost/backup-restore:test", "us", 100)
	backup := &model.InstallationBackup{
		ID:             "backup-1",
		InstallationID: "installation-1",
	}

	t.Run("no error when job does not exist", func(t *testing.T) {
		k8sClient := fake.NewSimpleClientset()
		jobClinet := k8sClient.BatchV1().Jobs("installation-1")

		err := operator.CleanupBackup(jobClinet, backup, logrus.New())
		require.NoError(t, err)
	})

	t.Run("delete job when exists", func(t *testing.T) {
		existing := &batchv1.Job{
			ObjectMeta: metav1.ObjectMeta{Name: "database-backup-backup-1", Namespace: "installation-1"},
			Status:     batchv1.JobStatus{Active: 1},
		}
		k8sClient := fake.NewSimpleClientset(existing)
		jobClinet := k8sClient.BatchV1().Jobs("installation-1")

		err := operator.CleanupBackup(jobClinet, backup, logrus.New())
		require.NoError(t, err)

		_, err = jobClinet.Get(context.Background(), "database-backup-backup-1", metav1.GetOptions{})
		require.Error(t, err)
		assert.True(t, k8sErrors.IsNotFound(err))
	})
}

func setJobActiveWhenExists(t *testing.T, client v1.JobInterface, name string) {
	ctx := context.Background()
	err := wait.Poll(1*time.Second, 30*time.Second, func() (bool, error) {
		job, err := client.Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return false, nil
		}
		job.Status.Active = 1
		_, err = client.Update(ctx, job, metav1.UpdateOptions{})
		if err != nil {
			return false, nil
		}
		return true, err
	})
	require.NoError(t, err)
}

func assertEnvVarEqual(t *testing.T, name, val string, env []corev1.EnvVar) {
	for _, e := range env {
		if e.Name == name {
			assert.Equal(t, e.Value, val)
			return
		}
	}

	assert.Fail(t, fmt.Sprintf("failed to find env var %s", name))
}

func assertEnvVarFromSecret(t *testing.T, name, secret, key string, env []corev1.EnvVar) {
	for _, e := range env {
		if e.Name == name {
			valFrom := e.ValueFrom
			require.NotNil(t, valFrom)
			require.NotNil(t, valFrom.SecretKeyRef)
			assert.Equal(t, secret, valFrom.SecretKeyRef.Name)
			assert.Equal(t, key, valFrom.SecretKeyRef.Key)
			return
		}
	}

	assert.Fail(t, fmt.Sprintf("failed to find env var %s", name))
}
