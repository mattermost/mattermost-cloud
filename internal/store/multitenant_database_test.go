package store

import (
	"testing"

	"github.com/mattermost/mattermost-cloud/internal/testlib"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/stretchr/testify/suite"
)

// TestMultitenantDatabase supplies tests for aws package Client.
type TestMultitenantDatabaseSuite struct {
	suite.Suite

	t *testing.T

	installationID0 string
	installationID1 string
	installationID2 string
	installationID3 string
	installationID4 string

	database1 *model.MultitenantDatabase
	database2 *model.MultitenantDatabase

	sqlStore *SQLStore
}

func (s *TestMultitenantDatabaseSuite) SetupTest() {
	s.sqlStore = MakeTestSQLStore(s.t, testlib.MakeLogger(s.t))

	s.installationID0 = "intalllation_id0"
	s.installationID1 = "intalllation_id1"
	s.installationID2 = "intalllation_id2"
	s.installationID3 = "intalllation_id3"
	s.installationID4 = "intalllation_id4"

	s.database1 = &model.MultitenantDatabase{
		ID: "database_id0",
	}

	s.database2 = &model.MultitenantDatabase{
		ID: "database_id1",
	}

	s.database1.SetInstallationIDs(model.MultitenantDatabaseInstallationIDs{s.installationID0, s.installationID1})
	s.database2.SetInstallationIDs(model.MultitenantDatabaseInstallationIDs{s.installationID2, s.installationID3, s.installationID4})

	err := s.sqlStore.CreateMultitenantDatabase(s.database1)
	s.Assert().NoError(err)

	err = s.sqlStore.CreateMultitenantDatabase(s.database2)
	s.Assert().NoError(err)

}

func TestMultitenantDatabase(t *testing.T) {
	suite.Run(t, &TestMultitenantDatabaseSuite{t: t})
}

func (s *TestMultitenantDatabaseSuite) TestCreateMultitenantDatabase() {
	db := model.MultitenantDatabase{
		ID: model.NewID(),
	}
	err := s.sqlStore.CreateMultitenantDatabase(&db)
	s.Assert().NoError(err)

	err = s.sqlStore.CreateMultitenantDatabase(&db)
	s.Assert().Error(err)
}

func (s *TestMultitenantDatabaseSuite) TestGet() {
	database, err := s.sqlStore.GetMultitenantDatabase(s.database1.ID)
	s.Assert().NoError(err)
	s.Assert().NotNil(database)
	s.Assert().Equal(*s.database1, *database)
}

func (s *TestMultitenantDatabaseSuite) TestGetLimitConstraint() {
	databases, err := s.sqlStore.GetMultitenantDatabases(&model.MultitenantDatabaseFilter{
		NumOfInstallationsLimit: 2,
		PerPage:                 model.AllPerPage,
	})
	s.Assert().NoError(err)
	s.Assert().NotNil(databases)
	s.Assert().Equal(1, len(databases))
	s.Assert().Equal(s.database1.ID, databases[0].ID)
}

func (s *TestMultitenantDatabaseSuite) TestGetLimitConstraintZero() {
	databases, err := s.sqlStore.GetMultitenantDatabases(&model.MultitenantDatabaseFilter{
		NumOfInstallationsLimit: 1,
		PerPage:                 model.AllPerPage,
	})
	s.Assert().NoError(err)
	s.Assert().NotNil(databases)
	s.Assert().Equal(0, len(databases))
}

func (s *TestMultitenantDatabaseSuite) TestGetLimitConstraintFilterNotNil() {
	databases, err := s.sqlStore.GetMultitenantDatabases(&model.MultitenantDatabaseFilter{
		NumOfInstallationsLimit: 0,
		PerPage:                 model.AllPerPage,
	})
	s.Assert().NoError(err)
	s.Assert().NotNil(databases)
	s.Assert().Equal(0, len(databases))
}

func (s *TestMultitenantDatabaseSuite) TestGetLimitConstraintAll() {
	db := model.MultitenantDatabase{
		ID: "database_id5",
	}
	err := s.sqlStore.CreateMultitenantDatabase(&db)
	s.Assert().NoError(err)

	databases, err := s.sqlStore.GetMultitenantDatabases(&model.MultitenantDatabaseFilter{
		NumOfInstallationsLimit: model.NoInstallationsLimit,
		PerPage:                 model.AllPerPage,
	})
	s.Assert().NoError(err)
	s.Assert().NotNil(databases)
	s.Assert().Equal(3, len(databases))
}

func (s *TestMultitenantDatabaseSuite) TestGetNoLimitConstraint() {
	databases, err := s.sqlStore.GetMultitenantDatabases(&model.MultitenantDatabaseFilter{
		PerPage: model.AllPerPage,
	})
	s.Assert().NoError(err)
	s.Assert().NotNil(databases)
	s.Assert().Equal(0, len(databases))
}
