// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package store

import (
	"strings"
	"testing"

	"github.com/mattermost/mattermost-cloud/internal/testlib"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// This test verifies various edge cases of joining Installation
// and InstallationDNS tables.
// Currently, not all of them can occur due to other business
// requirements, however, it is better not to be surprised if
// we decide to change something.
func Test_QueryInstallationsWithDNS(t *testing.T) {
	logger := testlib.MakeLogger(t)
	sqlStore := MakeTestSQLStore(t, logger)
	defer CloseConnection(t, sqlStore)

	installation1 := &model.Installation{
		Name: "multi-dns",
	}
	dnsRecords1 := []*model.InstallationDNS{
		{DomainName: "multi-dns1.com"},
		{DomainName: "multi-dns2.com"},
	}
	err := sqlStore.CreateInstallation(installation1, nil, dnsRecords1)
	require.NoError(t, err)

	installation2 := &model.Installation{
		Name: "multi-dns2",
	}
	dnsRecords2 := []*model.InstallationDNS{
		{DomainName: "multi1.com"},
		{DomainName: "multi2.com"},
	}
	err = sqlStore.CreateInstallation(installation2, nil, dnsRecords2)
	require.NoError(t, err)

	installation3 := &model.Installation{
		Name: "no-dns",
	}
	err = sqlStore.CreateInstallation(installation3, nil, nil)
	require.NoError(t, err)

	t.Run("query correct Installation by DNS", func(t *testing.T) {
		fetched, errTest := sqlStore.GetInstallations(&model.InstallationFilter{DNS: "multi-dns1.com", Paging: model.AllPagesNotDeleted()}, false, false)
		require.NoError(t, errTest)
		assert.Equal(t, installation1, fetched[0])

		fetched, errTest = sqlStore.GetInstallations(&model.InstallationFilter{DNS: "multi2.com", Paging: model.AllPagesNotDeleted()}, false, false)
		require.NoError(t, errTest)
		assert.Equal(t, installation2, fetched[0])
	})

	t.Run("return 0 installations if IDs and DNS do not match", func(t *testing.T) {
		fetched, errTest := sqlStore.GetInstallations(&model.InstallationFilter{InstallationIDs: []string{installation1.ID}, DNS: "multi2.com", Paging: model.AllPagesNotDeleted()}, false, false)
		require.NoError(t, errTest)
		assert.Equal(t, 0, len(fetched))
	})

	t.Run("return all installation without duplicates", func(t *testing.T) {
		fetched, errTest := sqlStore.GetInstallations(&model.InstallationFilter{Paging: model.AllPagesNotDeleted()}, false, false)
		require.NoError(t, errTest)
		assert.Equal(t, 3, len(fetched))
	})

	// Delete DNS
	err = sqlStore.DeleteInstallationDNS(installation2.ID, "multi1.com")
	require.NoError(t, err)

	t.Run("return 0 installation if not including delete and DNS deleted", func(t *testing.T) {
		fetched, errTest := sqlStore.GetInstallations(&model.InstallationFilter{DNS: "multi1.com", Paging: model.AllPagesNotDeleted()}, false, false)
		require.NoError(t, errTest)
		assert.Equal(t, 0, len(fetched))
	})
	t.Run("return installation when fetching by deleted DNS and should include deleted", func(t *testing.T) {
		fetched, errTest := sqlStore.GetInstallations(&model.InstallationFilter{DNS: "multi1.com", Paging: model.AllPagesWithDeleted()}, false, false)
		require.NoError(t, errTest)
		assert.Equal(t, installation2, fetched[0])
	})
	t.Run("return all installation without duplicates after DNS deletion", func(t *testing.T) {
		fetched, errTest := sqlStore.GetInstallations(&model.InstallationFilter{Paging: model.AllPagesNotDeleted()}, false, false)
		require.NoError(t, errTest)
		assert.Equal(t, 3, len(fetched))
	})

	// Delete Installation
	err = sqlStore.DeleteInstallation(installation2.ID)
	require.NoError(t, err)
	installation2, err = sqlStore.GetInstallation(installation2.ID, false, false)
	require.NoError(t, err)

	t.Run("return 0 installation if not including deleted and Installation deleted", func(t *testing.T) {
		fetched, err := sqlStore.GetInstallations(&model.InstallationFilter{DNS: "multi2.com", Paging: model.AllPagesNotDeleted()}, false, false)
		require.NoError(t, err)
		assert.Equal(t, 0, len(fetched))
	})
	t.Run("return installation if including deleted and Installation and DNS deleted", func(t *testing.T) {
		fetched, err := sqlStore.GetInstallations(&model.InstallationFilter{DNS: "multi1.com", Paging: model.AllPagesWithDeleted()}, false, false)
		require.NoError(t, err)
		assert.Equal(t, installation2, fetched[0])
	})
	t.Run("return all installation", func(t *testing.T) {
		fetched, err := sqlStore.GetInstallations(&model.InstallationFilter{Paging: model.AllPagesWithDeleted()}, false, false)
		require.NoError(t, err)
		assert.Equal(t, 3, len(fetched))

		fetched, err = sqlStore.GetInstallations(&model.InstallationFilter{Paging: model.AllPagesNotDeleted()}, false, false)
		require.NoError(t, err)
		assert.Equal(t, 2, len(fetched))
	})
}

func TestInstallationDNS(t *testing.T) {
	logger := testlib.MakeLogger(t)
	sqlStore := MakeTestSQLStore(t, logger)
	defer CloseConnection(t, sqlStore)

	installation := &model.Installation{
		Name: "test",
	}
	dnsRecords := []*model.InstallationDNS{
		{DomainName: "test.dns.com", IsPrimary: true},
	}

	err := sqlStore.CreateInstallation(installation, nil, dnsRecords)
	require.NoError(t, err)
	initialDNSRec := dnsRecords[0]

	t.Run("fail to create second primary DNS", func(t *testing.T) {
		dnsRecord := &model.InstallationDNS{DomainName: "test.dns-2.com", IsPrimary: true}
		errTest := sqlStore.AddInstallationDomain(installation, dnsRecord)
		require.Error(t, errTest)
		assert.Contains(t, strings.ToLower(errTest.Error()), "unique constraint") // Make sure error comes from DB
	})

	// Add DNS record
	addedRecord := &model.InstallationDNS{DomainName: "test.dns2.com"}
	err = sqlStore.AddInstallationDomain(installation, addedRecord)
	require.NoError(t, err)

	installationDTO, err := sqlStore.GetInstallationDTO(installation.ID, false, false)
	require.NoError(t, err)
	assert.Equal(t, 2, len(installationDTO.DNSRecords))

	installations, err := sqlStore.GetInstallations(&model.InstallationFilter{DNS: "test.dns.com", Paging: model.AllPagesNotDeleted()}, false, false)
	require.NoError(t, err)
	assert.Equal(t, installationDTO.Installation, installations[0])

	// Add new domain and ensure it is not primary.
	thirdRecord := &model.InstallationDNS{DomainName: "test.dns3.com"}
	err = sqlStore.AddInstallationDomain(installation, thirdRecord)
	require.NoError(t, err)

	records, err := sqlStore.GetDNSRecordsForInstallation(installation.ID)
	require.NoError(t, err)
	assert.Equal(t, true, records[0].IsPrimary)
	assert.Equal(t, false, records[1].IsPrimary)
	assert.Equal(t, false, records[2].IsPrimary)
	assert.Equal(t, initialDNSRec.ID, records[0].ID) // Sanity check

	// Switch primary domain name - IsPrimary should be set to false on others.
	err = sqlStore.SwitchPrimaryInstallationDomain(installation.ID, thirdRecord.ID)
	require.NoError(t, err)
	records, err = sqlStore.GetDNSRecordsForInstallation(installation.ID)
	require.NoError(t, err)
	assert.Equal(t, false, records[0].IsPrimary)
	assert.Equal(t, false, records[1].IsPrimary)
	assert.Equal(t, true, records[2].IsPrimary)
	assert.Equal(t, thirdRecord.ID, records[2].ID) // Sanity check

	record, err := sqlStore.GetInstallationDNS(thirdRecord.ID)
	require.NoError(t, err)
	assert.True(t, record.IsPrimary)

	// Delete InstallationDNS
	err = sqlStore.DeleteInstallationDNS(installation.ID, "test.dns.com")
	require.NoError(t, err)

	installations, err = sqlStore.GetInstallations(&model.InstallationFilter{DNS: "test.dns.com", Paging: model.AllPagesNotDeleted()}, false, false)
	require.NoError(t, err)
	assert.Equal(t, 0, len(installations))
}
