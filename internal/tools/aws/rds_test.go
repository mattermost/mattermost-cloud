package aws

import (
	"fmt"

	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/mock"
)

func (a *AWSTestSuite) TestCreateDatabaseSnapshot() {
	a.SetCreateDBClusterSnapshotExpectation(a.InstallationA.ID).Return(&rds.CreateDBClusterSnapshotOutput{}, nil).Once()

	err := a.Mocks.AWS.CreateDatabaseSnapshot(a.InstallationA.ID)

	a.Assert().NoError(err)
	a.Mocks.API.RDS.AssertExpectations(a.T())
}

func (a *AWSTestSuite) TestCreateDatabaseSnapshotError() {
	a.SetCreateDBClusterSnapshotExpectation(a.InstallationA.ID).Return(nil, errors.New("database is not stable")).Once()

	err := a.Mocks.AWS.CreateDatabaseSnapshot(a.InstallationA.ID)

	a.Assert().Error(err)
	a.Assert().Equal("failed to create a DB cluster snapshot for replication: database is not stable", err.Error())
	a.Mocks.API.RDS.AssertExpectations(a.T())
}

func RDSSnapshotTagValue(installationID string) string {
	return fmt.Sprintf("rds-snapshot-cloud-%s", installationID)
}

func (a *AWSTestSuite) SetCreateDBClusterSnapshotExpectation(installationID string) *mock.Call {
	return a.Mocks.API.RDS.On("CreateDBClusterSnapshot", mock.MatchedBy(func(input *rds.CreateDBClusterSnapshotInput) bool {
		return *input.DBClusterIdentifier == CloudID(installationID) &&
			*input.Tags[0].Key == DefaultClusterInstallationSnapshotTagKey &&
			*input.Tags[0].Value == RDSSnapshotTagValue(installationID)
	}))
}
