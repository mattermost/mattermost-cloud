package supervisor_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	awatModel "github.com/mattermost/awat/model"
	awatMocks "github.com/mattermost/awat/testlib/mocks"
	mocks "github.com/mattermost/mattermost-cloud/internal/mocks/aws-tools"
	"github.com/mattermost/mattermost-cloud/internal/supervisor"
	"github.com/mattermost/mattermost-cloud/internal/testlib"
	"github.com/mattermost/mattermost-cloud/model"
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

	testImport := func(failImport bool) error {
		awatClient := awatMocks.NewMockClient(gmctrl)
		store := new(mockInstallationStore)
		aws := mocks.NewMockAWS(gmctrl)
		importSupervisor := supervisor.NewImportSupervisor(
			aws,
			awatClient,
			store,
			&mockImportProvisioner{Fail: failImport},
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

		awatClient.EXPECT().
			CompleteImport(
				&awatModel.ImportCompletedWorkRequest{
					ID:    "some-import-id",
					Error: "",
				})

		aws.EXPECT().
			GetMultitenantBucketNameForInstallation(installationID, store).
			Return(destBucket, nil)

		aws.EXPECT().
			S3LargeCopy(&sourceBucket, &inputArchive, &destBucket,
				gomock.Any())

		return importSupervisor.Do()
	}

	t.Run("one installation pending work", func(t *testing.T) {
		err := testImport(false)
		assert.NoError(t, err, "error supervising")
	})

	t.Run("something goes wrong on import", func(t *testing.T) {
		err := testImport(true)
		assert.Error(t, err, "no error supervising")
	})
}

type mockImportProvisioner struct {
	Fail bool
}

func (m *mockImportProvisioner) ExecClusterInstallationCLI(cluster *model.Cluster, clusterInstallation *model.ClusterInstallation, args ...string) ([]byte, error) {
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
								"data": {}
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
								"progress": 0,
								"data": {}
						}
				]
		`), nil
		}
	}
}
