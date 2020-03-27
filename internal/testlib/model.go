package testlib

import (
	"github.com/golang/mock/gomock"
	mocks "github.com/mattermost/mattermost-cloud/internal/mocks/model"
)

// ModelMockedAPI has all mocked interfaces defined in model.
type ModelMockedAPI struct {
	DatabaseInstallationStore *mocks.MockInstallationDatabaseStoreInterface
}

// NewModelMockedAPI returns an instance of ModelMockedAPI.
func NewModelMockedAPI(ctrl *gomock.Controller) *ModelMockedAPI {
	return &ModelMockedAPI{
		DatabaseInstallationStore: mocks.NewMockInstallationDatabaseStoreInterface(ctrl),
	}
}
