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
			assert.Equal(t, "backup", createdJob.Labels["action"])
			assert.Equal(t, "installation-1", createdJob.Namespace)
			assert.Equal(t, backupBackoffLimit, *createdJob.Spec.BackoffLimit)
			assert.Equal(t, int32(100), *createdJob.Spec.TTLSecondsAfterFinished)

			podTemplate := createdJob.Spec.Template
			assert.Equal(t, "backup-restore", podTemplate.Labels["app"])
			assert.Equal(t, "backup", podTemplate.Labels["action"])
			assert.Equal(t, "mattermost/backup-restore:test", podTemplate.Spec.Containers[0].Image)

			assert.Equal(t, []string{"backup"}, podTemplate.Spec.Containers[0].Args)

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

		operator2 := NewBackupOperator("image", "us", -1)

		_, err := operator2.TriggerBackup(
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

// Tests CheckBackupStatus and CheckRestoreStatus as their logic is almost exactly the same
func TestOperator_CheckJobStatus(t *testing.T) {
	backupMeta := &model.InstallationBackup{
		ID:             "backup-1",
		InstallationID: "installation-1",
		State:          model.InstallationBackupStateBackupRequested,
	}

	k8sClient := fake.NewSimpleClientset()
	jobClient := k8sClient.BatchV1().Jobs("installation-1")

	operator := NewBackupOperator("mattermost/backup-restore:test", "us", 0)

	startTime := metav1.NewTime(time.Now().Add(time.Minute))
	endTime := metav1.NewTime(time.Now().Add(time.Hour))

	for _, testCase := range []struct {
		description       string
		checkFunc         func(jobsClient v1.JobInterface, backup *model.InstallationBackup, logger logrus.FieldLogger) (int64, error)
		jobName           string
		successStatus     batchv1.JobStatus
		expectedTimestamp int64
	}{
		{
			description:       "check backup status",
			checkFunc:         operator.CheckBackupStatus,
			jobName:           "database-backup-backup-1",
			successStatus:     batchv1.JobStatus{Succeeded: 1, StartTime: &startTime},
			expectedTimestamp: asMillis(startTime),
		},
		{
			description:       "check restore status",
			checkFunc:         operator.CheckRestoreStatus,
			jobName:           "database-restore-backup-1",
			successStatus:     batchv1.JobStatus{Succeeded: 1, CompletionTime: &endTime},
			expectedTimestamp: asMillis(endTime),
		},
	} {
		t.Run(testCase.description, func(t *testing.T) {
			t.Run("error when job does not exists", func(t *testing.T) {
				_, err := testCase.checkFunc(jobClient, backupMeta, logrus.New())
				require.Error(t, err)
			})

			job := &batchv1.Job{
				ObjectMeta: metav1.ObjectMeta{Name: testCase.jobName},
				Spec:       batchv1.JobSpec{},
				Status:     batchv1.JobStatus{},
			}
			var err error
			job, err = jobClient.Create(context.Background(), job, metav1.CreateOptions{})
			require.NoError(t, err)

			t.Run("return -1 start time if not finished", func(t *testing.T) {
				timestamp, err2 := testCase.checkFunc(jobClient, backupMeta, logrus.New())
				require.NoError(t, err2)
				assert.Equal(t, int64(-1), timestamp)
			})

			job.Status.Failed = backupBackoffLimit + 1
			job, err = jobClient.Update(context.Background(), job, metav1.UpdateOptions{})
			require.NoError(t, err)

			t.Run("ErrJobBackoffLimitReached when failed enough times", func(t *testing.T) {
				_, err = testCase.checkFunc(jobClient, backupMeta, logrus.New())
				require.Error(t, err)
				assert.Equal(t, ErrJobBackoffLimitReached, err)
			})

			job.Status = testCase.successStatus
			_, err = jobClient.Update(context.Background(), job, metav1.UpdateOptions{})
			require.NoError(t, err)

			t.Run("return timestamp when succeeded", func(t *testing.T) {
				timestamp, err := testCase.checkFunc(jobClient, backupMeta, logrus.New())
				require.NoError(t, err)
				assert.Equal(t, testCase.expectedTimestamp, timestamp)
			})
		})
	}
}

// Tests CleanupBackupJob and CleanupRestoreJob as their logic is almost exactly the same
func TestBackupOperator_CleanupJob(t *testing.T) {
	operator := NewBackupOperator("mattermost/backup-restore:test", "us", 100)
	backup := &model.InstallationBackup{
		ID:             "backup-1",
		InstallationID: "installation-1",
	}

	for _, testCase := range []struct {
		description string
		cleanupFunc func(jobsClient v1.JobInterface, backup *model.InstallationBackup, logger logrus.FieldLogger) error
		jobName     string
	}{
		{
			description: "cleanup backup job",
			cleanupFunc: operator.CleanupBackupJob,
			jobName:     "database-backup-backup-1",
		},
		{
			description: "cleanup restore job",
			cleanupFunc: operator.CleanupRestoreJob,
			jobName:     "database-restore-backup-1",
		},
	} {
		t.Run(testCase.description, func(t *testing.T) {
			t.Run("no error when job does not exist", func(t *testing.T) {
				k8sClient := fake.NewSimpleClientset()
				jobClinet := k8sClient.BatchV1().Jobs("installation-1")

				err := testCase.cleanupFunc(jobClinet, backup, logrus.New())
				require.NoError(t, err)
			})

			t.Run("delete job when exists", func(t *testing.T) {
				existing := &batchv1.Job{
					ObjectMeta: metav1.ObjectMeta{Name: testCase.jobName, Namespace: "installation-1"},
					Status:     batchv1.JobStatus{Active: 1},
				}
				k8sClient := fake.NewSimpleClientset(existing)
				jobClient := k8sClient.BatchV1().Jobs("installation-1")

				err := testCase.cleanupFunc(jobClient, backup, logrus.New())
				require.NoError(t, err)

				_, err = jobClient.Get(context.Background(), testCase.jobName, metav1.GetOptions{})
				require.Error(t, err)
				assert.True(t, k8sErrors.IsNotFound(err))
			})
		})
	}
}

func TestOperator_TriggerRestore(t *testing.T) {
	fileStoreSecret := "file-store-secret"
	fileStoreCfg := &model.FilestoreConfig{
		Secret: fileStoreSecret,
	}
	databaseSecret := "database-secret"

	backupMeta := &model.InstallationBackup{
		ID:             "backup-rest-1",
		InstallationID: "installation-1",
		State:          model.InstallationBackupStateBackupRequested,
		DataResidence: &model.S3DataResidence{
			Region:     "us-east",
			URL:        "filestore.com",
			Bucket:     "plastic",
			PathPrefix: "my-deer",
			ObjectKey:  "backup-backup-rest-1",
		},
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
				"BRT_STORAGE_TLS":  "false",
				"BRT_STORAGE_TYPE": "bifrost",
			},
		},
	} {
		t.Run(testCase.description, func(t *testing.T) {
			k8sClient := fake.NewSimpleClientset()
			jobClinet := k8sClient.BatchV1().Jobs("installation-1")
			go setJobActiveWhenExists(t, jobClinet, "database-restore-backup-rest-1")

			err := operator.TriggerRestore(
				jobClinet,
				backupMeta,
				testCase.installation,
				fileStoreCfg,
				databaseSecret,
				logrus.New())
			require.NoError(t, err)

			createdJob, err := jobClinet.Get(context.Background(), "database-restore-backup-rest-1", metav1.GetOptions{})
			require.NoError(t, err)

			assert.Equal(t, "backup-restore", createdJob.Labels["app"])
			assert.Equal(t, "restore", createdJob.Labels["action"])
			assert.Equal(t, "installation-1", createdJob.Namespace)
			assert.Equal(t, restoreBackoffLimit, *createdJob.Spec.BackoffLimit)
			assert.Equal(t, int32(100), *createdJob.Spec.TTLSecondsAfterFinished)

			podTemplate := createdJob.Spec.Template
			assert.Equal(t, "backup-restore", podTemplate.Labels["app"])
			assert.Equal(t, "restore", podTemplate.Labels["action"])
			assert.Equal(t, "mattermost/backup-restore:test", podTemplate.Spec.Containers[0].Image)

			assert.Equal(t, []string{"restore"}, podTemplate.Spec.Containers[0].Args)

			envs := podTemplate.Spec.Containers[0].Env
			assertEnvVarEqual(t, "BRT_STORAGE_REGION", "us-east", envs)
			assertEnvVarEqual(t, "BRT_STORAGE_BUCKET", "plastic", envs)
			assertEnvVarEqual(t, "BRT_STORAGE_ENDPOINT", testCase.expectedFileStoreURL, envs)
			assertEnvVarEqual(t, "BRT_STORAGE_OBJECT_KEY", "backup-backup-rest-1", envs)
			assertEnvVarEqual(t, "BRT_STORAGE_PATH_PREFIX", "my-deer", envs)
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
			ObjectMeta: metav1.ObjectMeta{Name: "database-restore-backup-rest-1", Namespace: "installation-1"},
			Status:     batchv1.JobStatus{Active: 1},
		}
		k8sClient := fake.NewSimpleClientset(existing)
		jobClinet := k8sClient.BatchV1().Jobs("installation-1")

		installation := &model.Installation{ID: "installation-1", Filestore: model.InstallationFilestoreMultiTenantAwsS3}

		err := operator.TriggerRestore(
			jobClinet,
			backupMeta,
			installation,
			fileStoreCfg,
			databaseSecret,
			logrus.New())
		require.NoError(t, err)
	})

	t.Run("set ttl to nil if negative value", func(t *testing.T) {
		k8sClient := fake.NewSimpleClientset()
		jobClinet := k8sClient.BatchV1().Jobs("installation-1")
		go setJobActiveWhenExists(t, jobClinet, "database-restore-backup-rest-1")

		installation := &model.Installation{ID: "installation-1", Filestore: model.InstallationFilestoreMultiTenantAwsS3}

		operator2 := NewBackupOperator("image", "us", -1)

		err := operator2.TriggerRestore(
			jobClinet,
			backupMeta,
			installation,
			fileStoreCfg,
			databaseSecret,
			logrus.New())
		require.NoError(t, err)

		createdJob, err := jobClinet.Get(context.Background(), "database-restore-backup-rest-1", metav1.GetOptions{})
		require.NoError(t, err)
		assert.Nil(t, createdJob.Spec.TTLSecondsAfterFinished)
	})

	t.Run("fail if job not ready", func(t *testing.T) {
		k8sClient := fake.NewSimpleClientset()
		jobClinet := k8sClient.BatchV1().Jobs("installation-1")
		installation := &model.Installation{ID: "installation-1", Filestore: model.InstallationFilestoreMultiTenantAwsS3}

		err := operator.TriggerRestore(
			jobClinet,
			backupMeta,
			installation,
			fileStoreCfg,
			databaseSecret,
			logrus.New())
		require.Error(t, err)
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
