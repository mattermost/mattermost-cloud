package testlib

import (
	mocks "github.com/mattermost/mattermost-cloud/internal/mocks/model"
)

// ModelMockedAPI has all mocked interfaces defined in model.
type ModelMockedAPI struct {
	DatabaseInstallationStore *mocks.InstallationDatabaseStoreInterface
}

// NewModelMockedAPI returns an instance of ModelMockedAPI.
func NewModelMockedAPI() *ModelMockedAPI {
	return &ModelMockedAPI{
		DatabaseInstallationStore: &mocks.InstallationDatabaseStoreInterface{},
	}
}
