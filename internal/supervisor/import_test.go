// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package supervisor_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	awatModel "github.com/mattermost/awat/model"
	awatMocks "github.com/mattermost/mattermost-cloud/internal/mocks/awat"
	awsMocks "github.com/mattermost/mattermost-cloud/internal/mocks/aws-tools"
	"github.com/mattermost/mattermost-cloud/internal/supervisor"
	"github.com/mattermost/mattermost-cloud/internal/testlib"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func TestImportSupervisor(t *testing.T) {
	logger := testlib.MakeLogger(t)
	gmctrl := gomock.NewController(t)
	defer gmctrl.Finish()
	var (
		translationID  string = "some-translation-id"
		importID       string = "some-import-id"
		installationID string = "some-installation-id"
		sourceBucket   string = "awat-bucket"
		destBucket     string = "the-multitenant-bucket"
		inputArchive   string = "user_upload.zip"

		resource string = fmt.Sprintf("%s/%s", sourceBucket, inputArchive)
	)

	t.Run("successfully import a translation", func(t *testing.T) {
		awatClient := awatMocks.NewMockAWATClient(gmctrl)
		store := new(mockInstallationStore)
		aws := awsMocks.NewMockAWS(gmctrl)
		importSupervisor := supervisor.NewImportSupervisor(
			aws,
			awatClient,
			store,
			&mockImportProvisioner{Fail: false},
			&mockEventProducer{},
			logger)

		awatClient.EXPECT().
			GetTranslationReadyToImport(gomock.Any()).
			Return(
				&awatModel.ImportStatus{
					Import: awatModel.Import{
						ID:            importID,
						CreateAt:      time.Now().UnixNano() / 1000,
						TranslationID: translationID,
						Resource:      resource,
					},
					InstallationID: installationID,
					Users:          30,
					Team:           "newteam",
					State:          "import-requested",
				}, nil)

		store.Installation = &model.Installation{
			ID:        installationID,
			Filestore: "bifrost",
			State:     "stable",
		}

		awatClient.EXPECT().
			GetImportStatusesByInstallation(installationID).
			Return([]*awatModel.ImportStatus{}, nil)

		awatClient.EXPECT().
			ReleaseLockOnImport(importID)

		aws.EXPECT().
			GetMultitenantBucketNameForInstallation(installationID, store).
			Return(destBucket, nil)

		aws.EXPECT().
			S3LargeCopy(&sourceBucket, &inputArchive, &destBucket,
				gomock.Any())

		err := importSupervisor.Do()
		assert.NoError(t, err, "error supervising")
	})

	t.Run("something goes wrong on import", func(t *testing.T) {
		awatClient := awatMocks.NewMockAWATClient(gmctrl)
		store := new(mockInstallationStore)
		aws := awsMocks.NewMockAWS(gmctrl)
		importSupervisor := supervisor.NewImportSupervisor(
			aws,
			awatClient,
			store,
			&mockImportProvisioner{Fail: true},
			&mockEventProducer{},
			logger)

		awatClient.EXPECT().
			GetTranslationReadyToImport(gomock.Any()).
			Return(
				&awatModel.ImportStatus{
					Import: awatModel.Import{
						ID:            importID,
						CreateAt:      time.Now().UnixNano() / 1000,
						TranslationID: translationID,
						Resource:      resource,
					},
					InstallationID: installationID,
					Users:          30,
					Team:           "newteam",
					State:          "import-requested",
				}, nil)

		store.Installation = &model.Installation{
			ID:        installationID,
			Filestore: "bifrost",
			State:     "stable",
		}

		awatClient.EXPECT().
			GetImportStatusesByInstallation(installationID).
			Return([]*awatModel.ImportStatus{}, nil)

		awatClient.EXPECT().
			ReleaseLockOnImport(importID)

		awatClient.EXPECT().
			CompleteImport(gomock.Any()).
			Return(nil)

		aws.EXPECT().
			GetMultitenantBucketNameForInstallation(installationID, store).
			Return(destBucket, nil)

		aws.EXPECT().
			S3LargeCopy(&sourceBucket, &inputArchive, &destBucket,
				gomock.Any())

		err := importSupervisor.Do()
		assert.Error(t, err, "no error supervising")
	})

	t.Run("no installations pending work", func(t *testing.T) {
		awatClient := awatMocks.NewMockAWATClient(gmctrl)
		store := new(mockInstallationStore)
		aws := awsMocks.NewMockAWS(gmctrl)
		importSupervisor := supervisor.NewImportSupervisor(
			aws,
			awatClient,
			store,
			&mockImportProvisioner{Fail: false},
			&mockEventProducer{},
			logger)

		awatClient.EXPECT().
			GetTranslationReadyToImport(gomock.Any()).
			Return(
				nil, nil)

		awatClient.EXPECT().
			GetImportStatusesByInstallation(gomock.Any()).
			Return([]*awatModel.ImportStatus{}, nil)

		err := importSupervisor.Do()
		assert.NoError(t, err, "error after no work found")
	})

	t.Run("handling an error from the AWAT", func(t *testing.T) {
		awatClient := awatMocks.NewMockAWATClient(gmctrl)
		store := new(mockInstallationStore)
		aws := awsMocks.NewMockAWS(gmctrl)
		importSupervisor := supervisor.NewImportSupervisor(
			aws,
			awatClient,
			store,
			&mockImportProvisioner{Fail: false},
			&mockEventProducer{},
			logger)

		awatClient.EXPECT().
			GetTranslationReadyToImport(gomock.Any()).
			Return(
				nil, errors.New("some error from AWAT"))

		awatClient.EXPECT().
			GetImportStatusesByInstallation(gomock.Any()).
			Return([]*awatModel.ImportStatus{}, nil)

		err := importSupervisor.Do()
		assert.Error(t, err, "expected failure due to error from AWAT")
	})

	t.Run("copying the file to the Installation S3 bucket fails", func(t *testing.T) {
		awatClient := awatMocks.NewMockAWATClient(gmctrl)
		store := new(mockInstallationStore)
		aws := awsMocks.NewMockAWS(gmctrl)
		importSupervisor := supervisor.NewImportSupervisor(
			aws,
			awatClient,
			store,
			&mockImportProvisioner{Fail: false},
			&mockEventProducer{},
			logger)

		awatClient.EXPECT().
			GetTranslationReadyToImport(gomock.Any()).
			Return(
				&awatModel.ImportStatus{
					Import: awatModel.Import{
						ID:            importID,
						CreateAt:      time.Now().UnixNano() / 1000,
						TranslationID: translationID,
						Resource:      resource,
					},
					InstallationID: installationID,
					Users:          30,
					Team:           "newteam",
					State:          "import-requested",
				}, nil)

		store.Installation = &model.Installation{
			ID:        installationID,
			Filestore: "bifrost",
			State:     "stable",
		}

		awatClient.EXPECT().
			ReleaseLockOnImport(importID)

		aws.EXPECT().
			GetMultitenantBucketNameForInstallation(installationID, store).
			Return(destBucket, nil)

		aws.EXPECT().
			S3LargeCopy(&sourceBucket, &inputArchive, &destBucket,
				gomock.Any()).
			Return(errors.New("some AWS error"))

		awatClient.EXPECT().
			GetImportStatusesByInstallation(installationID).
			Return([]*awatModel.ImportStatus{}, nil)

		awatClient.EXPECT().
			CompleteImport(gomock.Any()).
			Return(nil)

		err := importSupervisor.Do()
		time.Sleep(time.Second)
		// sleep to avoid a race that only occurs in testing because some
		// member of the MockAWATClient ceases to exist when tests are
		// done, but the Supervisor in the actual code exists forever
		assert.Error(t, err, "error not handled properly")
	})
}

type mockImportProvisioner struct {
	Fail bool
}

func (m *mockImportProvisioner) ExecMMCTL(cluster *model.Cluster, clusterInstallation *model.ClusterInstallation, args ...string) ([]byte, error) {
	switch args[len(args)-1] {
	case "TeamSettings.MaxUsersPerTeam":
		return []byte("10"), nil
	default:
		if m.Fail {
			return []byte(`
				[
						{
								"id": "wxp9rsparprwdnenjguhys61dy",
								"type": "import",
								"priority": 0,
								"create_at": 1619598759849,
								"start_at": 1619598771479,
								"last_activity_at": 1619598771485,
								"status": "error",
								"progress": 0,
								"data": {
										"error": "FUBAR",
										"line_number": "70",
										"import_file": "some-import-file.zip"
								}
						}
				]
		`), nil
		} else {
			return []byte(`
				[
						{
								"id": "wxp9rsparprwdnenjguhys61dy",
								"type": "import",
								"priority": 0,
								"create_at": 1619598759849,
								"start_at": 1619598771479,
								"last_activity_at": 1619598771485,
								"status": "success",
								"progress": 0
						}
				]
		`), nil
		}
	}
}
