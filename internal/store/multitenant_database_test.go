// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package store

import (
	"testing"
	"time"

	"github.com/mattermost/mattermost-cloud/internal/testlib"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/stretchr/testify/suite"
)

// TestMultitenantDatabase supplies tests for aws package Client.
type TestMultitenantDatabaseSuite struct {
	suite.Suite

	t *testing.T

	lockerID string

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

	s.lockerID = s.installationID0

	s.database1 = &model.MultitenantDatabase{
		ID:    "database_id0",
		VpcID: "vpc_id0",
	}

	s.database2 = &model.MultitenantDatabase{
		ID:    "database_id1",
		VpcID: "vpc_id1",
	}

	s.database1.Installations = model.MultitenantDatabaseInstallations{s.installationID0, s.installationID1}
	s.database2.Installations = model.MultitenantDatabaseInstallations{s.installationID2, s.installationID3, s.installationID4}

	err := s.sqlStore.CreateMultitenantDatabase(s.database1)
	s.Assert().NoError(err)

	err = s.sqlStore.CreateMultitenantDatabase(s.database2)
	s.Assert().NoError(err)

	time.Sleep(1 * time.Millisecond)
}

func TestMultitenantDatabase(t *testing.T) {
	suite.Run(t, &TestMultitenantDatabaseSuite{t: t})
}

func (s *TestMultitenantDatabaseSuite) TestCreate() {
	db := model.MultitenantDatabase{
		ID:    "database_some_id",
		VpcID: "database_vpc_id",
	}
	err := s.sqlStore.CreateMultitenantDatabase(&db)
	s.Assert().NoError(err)
	s.Assert().Greater(db.CreateAt, int64(0))

	err = s.sqlStore.CreateMultitenantDatabase(&db)
	s.Assert().Error(err)
}

func (s *TestMultitenantDatabaseSuite) TestCreateNilIDError() {
	db := model.MultitenantDatabase{}
	err := s.sqlStore.CreateMultitenantDatabase(&db)
	s.Assert().Error(err)
}

func (s *TestMultitenantDatabaseSuite) TestGet() {
	database, err := s.sqlStore.GetMultitenantDatabase(s.database1.ID)
	s.Assert().NoError(err)
	s.Assert().NotNil(database)
	s.Assert().Equal(*s.database1, *database)
}

func (s *TestMultitenantDatabaseSuite) TestGetNotFound() {
	database, err := s.sqlStore.GetMultitenantDatabase("not_found_id")
	s.Assert().NoError(err)
	s.Assert().Nil(database)
}

func (s *TestMultitenantDatabaseSuite) TestGetLimitConstraint() {
	databases, err := s.sqlStore.GetMultitenantDatabases(&model.MultitenantDatabaseFilter{
		MaxInstallationsLimit: 2,
		PerPage:               model.AllPerPage,
	})
	s.Assert().NoError(err)
	s.Assert().Equal(0, len(databases))
}

func (s *TestMultitenantDatabaseSuite) TestGetLimitConstraintOne() {
	databases, err := s.sqlStore.GetMultitenantDatabases(&model.MultitenantDatabaseFilter{
		MaxInstallationsLimit: 3,
		PerPage:               model.AllPerPage,
	})
	s.Assert().NoError(err)
	s.Assert().NotNil(databases)
	s.Assert().Equal(1, len(databases))
	s.Assert().Equal(s.database1.ID, databases[0].ID)
}

func (s *TestMultitenantDatabaseSuite) TestGetLimitConstraintZero() {
	databases, err := s.sqlStore.GetMultitenantDatabases(&model.MultitenantDatabaseFilter{
		MaxInstallationsLimit: 1,
		PerPage:               model.AllPerPage,
	})
	s.Assert().NoError(err)
	s.Assert().Equal(0, len(databases))
}

func (s *TestMultitenantDatabaseSuite) TestGetLimitConstraintFilterNotNil() {
	databases, err := s.sqlStore.GetMultitenantDatabases(&model.MultitenantDatabaseFilter{
		MaxInstallationsLimit: 0,
		PerPage:               model.AllPerPage,
	})
	s.Assert().NoError(err)
	s.Assert().Equal(0, len(databases))
}

func (s *TestMultitenantDatabaseSuite) TestGetLimitConstraintAll() {
	db := model.MultitenantDatabase{
		ID: "database_id5",
	}
	err := s.sqlStore.CreateMultitenantDatabase(&db)
	s.Assert().NoError(err)

	databases, err := s.sqlStore.GetMultitenantDatabases(&model.MultitenantDatabaseFilter{
		MaxInstallationsLimit: model.NoInstallationsLimit,
		PerPage:               model.AllPerPage,
	})
	s.Assert().NoError(err)
	s.Assert().NotNil(databases)
	s.Assert().Equal(3, len(databases))
}

func (s *TestMultitenantDatabaseSuite) TestGetLockerIDConstraintAll() {
	locked, err := s.sqlStore.LockMultitenantDatabase(s.database1.ID, s.installationID0)
	s.Assert().NoError(err)
	s.Assert().True(locked)

	databases, err := s.sqlStore.GetMultitenantDatabases(&model.MultitenantDatabaseFilter{
		LockerID:              s.installationID0,
		MaxInstallationsLimit: model.NoInstallationsLimit,
		PerPage:               model.AllPerPage,
	})
	s.Assert().NoError(err)
	s.Assert().NotNil(databases)
	s.Assert().Equal(1, len(databases))
	s.Assert().Equal(s.database1.ID, databases[0].ID)
}

func (s *TestMultitenantDatabaseSuite) TestGetNoLimitConstraint() {
	databases, err := s.sqlStore.GetMultitenantDatabases(&model.MultitenantDatabaseFilter{
		PerPage: model.AllPerPage,
	})
	s.Assert().NoError(err)
	s.Assert().Equal(0, len(databases))
}

func (s *TestMultitenantDatabaseSuite) TestGetMultitenantDatabase() {
	databases, err := s.sqlStore.GetMultitenantDatabases(&model.MultitenantDatabaseFilter{
		PerPage: model.AllPerPage,
	})
	s.Assert().NoError(err)
	s.Assert().Equal(0, len(databases))
}

func (s *TestMultitenantDatabaseSuite) TestUpdate() {
	locked, err := s.sqlStore.LockMultitenantDatabase(s.database1.ID, s.lockerID)
	s.Assert().NoError(err)
	s.Assert().True(locked)

	s.database1.Installations = model.MultitenantDatabaseInstallations{model.NewID()}
	s.database1.LockAcquiredBy = &s.lockerID

	err = s.sqlStore.UpdateMultitenantDatabase(s.database1)
	s.Assert().NoError(err)

	database, err := s.sqlStore.GetMultitenantDatabase(s.database1.ID)
	s.Assert().NoError(err)
	s.Assert().NotNil(database)
	s.Assert().Equal(s.database1.Installations, database.Installations)
}

func (s *TestMultitenantDatabaseSuite) TestGetMultitenantDatabaseForInstallationID() {
	database, err := s.sqlStore.GetMultitenantDatabaseForInstallationID(s.installationID0)
	s.Assert().NoError(err)
	s.Assert().NotNil(database)
	s.Assert().Equal(s.database1.ID, database.ID)
}

func (s *TestMultitenantDatabaseSuite) TestGetMultitenantDatabaseForInstallationIDErrorOne() {
	database, err := s.sqlStore.GetMultitenantDatabaseForInstallationID("banana")
	s.Assert().Error(err)
	s.Assert().Nil(database)
	s.Assert().Equal("expected exactly one multitenant database, but found 0", err.Error())
}

func (s *TestMultitenantDatabaseSuite) TestGetMultitenantDatabaseForInstallationIDErrorMany() {
	db := model.MultitenantDatabase{
		ID: "database_some_id",
	}
	db.Installations = model.MultitenantDatabaseInstallations{s.installationID0}
	err := s.sqlStore.CreateMultitenantDatabase(&db)
	s.Assert().NoError(err)

	database, err := s.sqlStore.GetMultitenantDatabaseForInstallationID(s.installationID0)
	s.Assert().Error(err)
	s.Assert().Nil(database)
	s.Assert().Equal("expected exactly one multitenant database, but found 2", err.Error())
}

func (s *TestMultitenantDatabaseSuite) TestGetDatabasesWithVpcIDFilter() {
	databases, err := s.sqlStore.GetMultitenantDatabases(&model.MultitenantDatabaseFilter{
		VpcID:                 "vpc_id0",
		MaxInstallationsLimit: model.NoInstallationsLimit,
		PerPage:               model.AllPerPage,
	})
	s.Assert().NoError(err)
	s.Assert().NotNil(databases)
	s.Assert().Equal(1, len(databases))

	databases, err = s.sqlStore.GetMultitenantDatabases(&model.MultitenantDatabaseFilter{
		VpcID:                 "does_not_exist",
		MaxInstallationsLimit: model.NoInstallationsLimit,
		PerPage:               model.AllPerPage,
	})
	s.Assert().NoError(err)
	s.Assert().Nil(databases)
	s.Assert().Equal(0, len(databases))
}
