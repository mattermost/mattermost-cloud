// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package api_test

import (
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/mattermost/mattermost-cloud/internal/api"
	"github.com/mattermost/mattermost-cloud/internal/store"
	"github.com/mattermost/mattermost-cloud/internal/testlib"
	"github.com/mattermost/mattermost-cloud/internal/testutil"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_AddInstallationDNS(t *testing.T) {
	logger := testlib.MakeLogger(t)
	sqlStore := store.MakeTestSQLStore(t, logger)
	defer store.CloseConnection(t, sqlStore)

	router := mux.NewRouter()
	api.Register(router, &api.Context{
		Store:         sqlStore,
		Supervisor:    &mockSupervisor{},
		EventProducer: testutil.SetupTestEventsProducer(sqlStore, logger),
		Logger:        logger,
	})
	ts := httptest.NewServer(router)
	defer ts.Close()
	model.SetDeployOperators(true, true)

	client := model.NewClient(ts.URL)

	// Create installation with multiple DNS right away.
	multiDNSInstallationReq := &model.CreateInstallationRequest{
		Name:     "multi-dns",
		DNSNames: []string{"multi-dns.example.com", "multi-dns.dns.com"},
		OwnerID:  "test",
	}
	installation1, err := client.CreateInstallation(multiDNSInstallationReq)
	require.NoError(t, err)
	assert.Equal(t, 2, len(installation1.DNSRecords))
	assert.Equal(t, "multi-dns.example.com", installation1.DNS)
	assert.Equal(t, true, installation1.DNSRecords[0].IsPrimary)

	// Assert one record set as primary.
	dnsRecords, err := sqlStore.GetDNSRecordsForInstallation(installation1.ID)
	require.NoError(t, err)
	assert.Equal(t, true, dnsRecords[0].IsPrimary)
	assert.Equal(t, "multi-dns.example.com", dnsRecords[0].DomainName)
	assert.Equal(t, false, dnsRecords[1].IsPrimary)

	t.Run("cannot create Installation with name not matching DNS", func(t *testing.T) {
		_, err = client.CreateInstallation(&model.CreateInstallationRequest{
			Name:    "test",
			DNS:     "not-test.dns.com",
			OwnerID: "test",
		})
		require.Error(t, err)
	})

	t.Run("cannot add already existing DNS", func(t *testing.T) {
		_, err = client.AddInstallationDNS(installation1.ID, &model.AddDNSRecordRequest{
			DNS: "multi-dns.example.com",
		})
		require.Error(t, err)
	})

	// Create installation using old API.
	oldAPIInstallation := &model.CreateInstallationRequest{
		DNS:     "old-api-dns.example.com",
		OwnerID: "test",
	}
	installation2, err := client.CreateInstallation(oldAPIInstallation)
	require.NoError(t, err)
	assert.Equal(t, 1, len(installation2.DNSRecords))
	assert.Equal(t, "old-api-dns.example.com", installation2.DNS)
	assert.Equal(t, "old-api-dns", installation2.Name)

	// Set installation to stable so we can add DNS.
	installation2.State = model.InstallationStateStable
	err = sqlStore.UpdateInstallation(installation2.Installation)
	require.NoError(t, err)

	installation2, err = client.AddInstallationDNS(installation2.ID, &model.AddDNSRecordRequest{
		DNS: "old-api-dns.dns.com",
	})
	require.NoError(t, err)
	installation2Fetched, err := client.GetInstallation(installation2.ID, nil)
	require.NoError(t, err)
	assert.Equal(t, installation2, installation2Fetched)

	t.Run("query installation by any DNS", func(t *testing.T) {
		for _, testCase := range []struct {
			dns          string
			installation *model.InstallationDTO
		}{
			{
				dns:          "multi-dns.example.com",
				installation: installation1,
			},
			{
				dns:          "multi-dns.dns.com",
				installation: installation1,
			},
			{
				dns:          "old-api-dns.example.com",
				installation: installation2,
			},
			{
				dns:          "old-api-dns.dns.com",
				installation: installation2,
			},
		} {
			t.Run(testCase.dns, func(t *testing.T) {
				fetched, err := client.GetInstallationByDNS(testCase.dns, nil)
				require.NoError(t, err)
				assert.Equal(t, testCase.installation, fetched)
			})
		}
	})
}

func Test_SetDomainNamePrimary(t *testing.T) {
	logger := testlib.MakeLogger(t)
	sqlStore := store.MakeTestSQLStore(t, logger)
	defer store.CloseConnection(t, sqlStore)

	router := mux.NewRouter()
	api.Register(router, &api.Context{
		Store:         sqlStore,
		Supervisor:    &mockSupervisor{},
		EventProducer: testutil.SetupTestEventsProducer(sqlStore, logger),
		Logger:        logger,
	})
	ts := httptest.NewServer(router)
	defer ts.Close()
	model.SetDeployOperators(true, true)

	client := model.NewClient(ts.URL)

	// Create installation with multiple DNS right away.
	multiDNSInstallationReq := &model.CreateInstallationRequest{
		Name:     "dns",
		DNSNames: []string{"dns.example.com", "dns.dns.com"},
		OwnerID:  "test",
	}
	installation, err := client.CreateInstallation(multiDNSInstallationReq)
	require.NoError(t, err)
	installation.State = model.InstallationStateStable
	err = sqlStore.UpdateInstallationState(installation.Installation)
	require.NoError(t, err)

	installation, err = client.AddInstallationDNS(installation.ID, &model.AddDNSRecordRequest{
		DNS: "dns.dns3.com",
	})
	require.NoError(t, err)
	// New domain name should not be primary.
	assert.Equal(t, false, installation.DNSRecords[2].IsPrimary)

	t.Run("cannot set non existing record as primary", func(t *testing.T) {
		_, err = client.SetInstallationDomainPrimary(installation.ID, "not-existing")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "404")
	})

	installation, err = client.SetInstallationDomainPrimary(installation.ID, installation.DNSRecords[2].ID)
	require.NoError(t, err)
	assert.Equal(t, true, installation.DNSRecords[2].IsPrimary)

	installationFetched, err := client.GetInstallation(installation.ID, nil)
	require.NoError(t, err)
	assert.Equal(t, installationFetched, installation)
}
