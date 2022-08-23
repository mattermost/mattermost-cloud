// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package api_test

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/mattermost/mattermost-cloud/internal/api"
	"github.com/mattermost/mattermost-cloud/internal/store"
	"github.com/mattermost/mattermost-cloud/internal/testlib"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/stretchr/testify/require"
)

func TestCreateWebhook(t *testing.T) {
	logger := testlib.MakeLogger(t)
	sqlStore := store.MakeTestSQLStore(t, logger)

	router := mux.NewRouter()
	api.Register(router, &api.Context{
		Store:      sqlStore,
		Supervisor: &mockSupervisor{},
		Metrics:    &mockMetrics{},
		Logger:     logger,
	})
	ts := httptest.NewServer(router)
	defer ts.Close()

	client := model.NewClient(ts.URL)

	t.Run("invalid payload", func(t *testing.T) {
		resp, err := http.Post(fmt.Sprintf("%s/api/webhooks", ts.URL), "application/json", bytes.NewReader([]byte("invalid")))
		require.NoError(t, err)
		require.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("empty payload", func(t *testing.T) {
		resp, err := http.Post(fmt.Sprintf("%s/api/webhooks", ts.URL), "application/json", bytes.NewReader([]byte("")))
		require.NoError(t, err)
		require.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("missing owner", func(t *testing.T) {
		_, err := client.CreateWebhook(&model.CreateWebhookRequest{
			URL: "https://validurl.com",
		})
		require.EqualError(t, err, "failed with status code 400")
	})

	t.Run("missing url", func(t *testing.T) {
		_, err := client.CreateWebhook(&model.CreateWebhookRequest{
			OwnerID: "owner",
		})
		require.EqualError(t, err, "failed with status code 400")
	})

	t.Run("invalid url", func(t *testing.T) {
		_, err := client.CreateWebhook(&model.CreateWebhookRequest{
			OwnerID: "owner",
			URL:     "htp://invalidurl.com",
		})
		require.EqualError(t, err, "failed with status code 400")
	})

	t.Run("valid", func(t *testing.T) {
		webhook, err := client.CreateWebhook(&model.CreateWebhookRequest{
			OwnerID: "owner",
			URL:     "https://validurl.com",
		})
		require.NoError(t, err)
		require.NotEmpty(t, webhook.ID)
		require.Equal(t, "owner", webhook.OwnerID)
		require.Equal(t, "https://validurl.com", webhook.URL)
		require.NotEqual(t, 0, webhook.CreateAt)
		require.EqualValues(t, 0, webhook.DeleteAt)
	})
}

func TestGetWebhooks(t *testing.T) {
	logger := testlib.MakeLogger(t)
	sqlStore := store.MakeTestSQLStore(t, logger)

	router := mux.NewRouter()
	api.Register(router, &api.Context{
		Store:      sqlStore,
		Supervisor: &mockSupervisor{},
		Metrics:    &mockMetrics{},
		Logger:     logger,
	})
	ts := httptest.NewServer(router)
	defer ts.Close()

	client := model.NewClient(ts.URL)

	t.Run("unknown webhook", func(t *testing.T) {
		webhook, err := client.GetWebhook(model.NewID())
		require.NoError(t, err)
		require.Nil(t, webhook)
	})

	t.Run("no webhooks", func(t *testing.T) {
		webhooks, err := client.GetWebhooks(&model.GetWebhooksRequest{
			Paging: model.AllPagesNotDeleted(),
		})
		require.NoError(t, err)
		require.Empty(t, webhooks)
	})

	t.Run("parameter handling", func(t *testing.T) {
		t.Run("invalid page", func(t *testing.T) {
			resp, err := http.Get(fmt.Sprintf("%s/api/webhooks?page=invalid&per_page=100", ts.URL))
			require.NoError(t, err)
			require.Equal(t, http.StatusBadRequest, resp.StatusCode)
		})

		t.Run("invalid perPage", func(t *testing.T) {
			resp, err := http.Get(fmt.Sprintf("%s/api/webhooks?page=0&per_page=invalid", ts.URL))
			require.NoError(t, err)
			require.Equal(t, http.StatusBadRequest, resp.StatusCode)
		})

		t.Run("no paging parameters", func(t *testing.T) {
			resp, err := http.Get(fmt.Sprintf("%s/api/webhooks", ts.URL))
			require.NoError(t, err)
			require.Equal(t, http.StatusOK, resp.StatusCode)
		})

		t.Run("missing page", func(t *testing.T) {
			resp, err := http.Get(fmt.Sprintf("%s/api/webhooks?per_page=100", ts.URL))
			require.NoError(t, err)
			require.Equal(t, http.StatusOK, resp.StatusCode)
		})

		t.Run("missing perPage", func(t *testing.T) {
			resp, err := http.Get(fmt.Sprintf("%s/api/webhooks?page=1", ts.URL))
			require.NoError(t, err)
			require.Equal(t, http.StatusOK, resp.StatusCode)
		})
	})

	t.Run("results", func(t *testing.T) {
		ownerID1 := model.NewID()
		ownerID2 := model.NewID()

		webhook1, err := client.CreateWebhook(&model.CreateWebhookRequest{
			OwnerID: ownerID1,
			URL:     "https://validurl1.com",
		})
		require.NoError(t, err)

		time.Sleep(1 * time.Millisecond)

		webhook2, err := client.CreateWebhook(&model.CreateWebhookRequest{
			OwnerID: ownerID2,
			URL:     "https://validurl2.com",
		})
		require.NoError(t, err)

		time.Sleep(1 * time.Millisecond)

		webhook3, err := client.CreateWebhook(&model.CreateWebhookRequest{
			OwnerID: ownerID1,
			URL:     "https://validurl3.com",
		})
		require.NoError(t, err)

		time.Sleep(1 * time.Millisecond)

		webhook4, err := client.CreateWebhook(&model.CreateWebhookRequest{
			OwnerID: ownerID2,
			URL:     "https://validurl4.com",
		})
		require.NoError(t, err)
		err = sqlStore.DeleteWebhook(webhook4.ID)
		require.NoError(t, err)
		webhook4, err = client.GetWebhook(webhook4.ID)
		require.NoError(t, err)

		t.Run("get webhook", func(t *testing.T) {
			t.Run("webhook 1", func(t *testing.T) {
				webhook, err := client.GetWebhook(webhook1.ID)
				require.NoError(t, err)
				require.Equal(t, webhook1, webhook)
			})

			t.Run("get deleted webhook", func(t *testing.T) {
				webhook, err := client.GetWebhook(webhook4.ID)
				require.NoError(t, err)
				require.Equal(t, webhook4, webhook)
			})
		})

		t.Run("get installations", func(t *testing.T) {
			testCases := []struct {
				Description        string
				GetWebhooksRequest *model.GetWebhooksRequest
				Expected           []*model.Webhook
			}{
				{
					"page 0, perPage 2, exclude deleted",
					&model.GetWebhooksRequest{
						Paging: model.Paging{
							Page:           0,
							PerPage:        2,
							IncludeDeleted: false,
						},
					},
					[]*model.Webhook{webhook1, webhook2},
				},

				{
					"page 1, perPage 2, exclude deleted",
					&model.GetWebhooksRequest{
						Paging: model.Paging{
							Page:           1,
							PerPage:        2,
							IncludeDeleted: false,
						},
					},
					[]*model.Webhook{webhook3},
				},

				{
					"page 0, perPage 2, include deleted",
					&model.GetWebhooksRequest{
						Paging: model.Paging{
							Page:           0,
							PerPage:        2,
							IncludeDeleted: true,
						},
					},
					[]*model.Webhook{webhook1, webhook2},
				},

				{
					"page 1, perPage 2, include deleted",
					&model.GetWebhooksRequest{
						Paging: model.Paging{
							Page:           1,
							PerPage:        2,
							IncludeDeleted: true,
						},
					},
					[]*model.Webhook{webhook3, webhook4},
				},
				{
					"filter by owner",
					&model.GetWebhooksRequest{
						OwnerID: ownerID1,
						Paging:  model.AllPagesNotDeleted(),
					},
					[]*model.Webhook{webhook1, webhook3},
				},
			}

			for _, testCase := range testCases {
				t.Run(testCase.Description, func(t *testing.T) {
					webhooks, err := client.GetWebhooks(testCase.GetWebhooksRequest)
					require.NoError(t, err)
					require.Equal(t, testCase.Expected, webhooks)
				})
			}
		})
	})
}

func TestDeleteWebhook(t *testing.T) {
	logger := testlib.MakeLogger(t)
	sqlStore := store.MakeTestSQLStore(t, logger)

	router := mux.NewRouter()
	api.Register(router, &api.Context{
		Store:      sqlStore,
		Supervisor: &mockSupervisor{},
		Metrics:    &mockMetrics{},
		Logger:     logger,
	})
	ts := httptest.NewServer(router)
	defer ts.Close()

	client := model.NewClient(ts.URL)

	webhook, err := client.CreateWebhook(&model.CreateWebhookRequest{
		OwnerID: "owner",
		URL:     "https://validurl.com",
	})
	require.NoError(t, err)

	t.Run("unknown webhook", func(t *testing.T) {
		err := client.DeleteWebhook(model.NewID())
		require.EqualError(t, err, "failed with status code 404")
	})

	t.Run("known webhook", func(t *testing.T) {
		err := client.DeleteWebhook(webhook.ID)
		require.NoError(t, err)
	})

	t.Run("already deleted webhook", func(t *testing.T) {
		err := client.DeleteWebhook(webhook.ID)
		require.Error(t, err)
	})

	t.Run("ensure webhook is deleted", func(t *testing.T) {
		webhook, err := client.GetWebhook(webhook.ID)
		require.NoError(t, err)
		require.True(t, webhook.IsDeleted())
	})
}
