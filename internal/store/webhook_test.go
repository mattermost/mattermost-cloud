// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package store

import (
	"testing"
	"time"

	"github.com/mattermost/mattermost-cloud/internal/testlib"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/stretchr/testify/require"
)

func TestWebhooks(t *testing.T) {
	t.Run("get unknown webhook", func(t *testing.T) {
		logger := testlib.MakeLogger(t)
		sqlStore := MakeTestSQLStore(t, logger)

		webhook, err := sqlStore.GetWebhook("unknown")
		require.NoError(t, err)
		require.Nil(t, webhook)
	})

	t.Run("get webhooks", func(t *testing.T) {
		logger := testlib.MakeLogger(t)
		sqlStore := MakeTestSQLStore(t, logger)

		webhook1 := &model.Webhook{
			OwnerID: "owner1",
			URL:     "https://url1.com",
			Headers: &model.StringMap{
				"Foo": "bar",
			},
		}

		webhook2 := &model.Webhook{
			OwnerID: "owner2",
			URL:     "https://url2.com",
			Headers: &model.StringMap{
				"Foo": "bar",
			},
		}

		err := sqlStore.CreateWebhook(webhook1)
		require.NoError(t, err)

		time.Sleep(1 * time.Millisecond)

		err = sqlStore.CreateWebhook(webhook2)
		require.NoError(t, err)

		actualWebhook1, err := sqlStore.GetWebhook(webhook1.ID)
		require.NoError(t, err)
		require.Equal(t, webhook1, actualWebhook1)

		actualWebhook2, err := sqlStore.GetWebhook(webhook2.ID)
		require.NoError(t, err)
		require.Equal(t, webhook2, actualWebhook2)

		actualWebhooks, err := sqlStore.GetWebhooks(&model.WebhookFilter{Paging: model.Paging{Page: 0, PerPage: 0, IncludeDeleted: false}})
		require.NoError(t, err)
		require.Empty(t, actualWebhooks)

		actualWebhooks, err = sqlStore.GetWebhooks(&model.WebhookFilter{Paging: model.Paging{Page: 0, PerPage: 1, IncludeDeleted: false}})
		require.NoError(t, err)
		require.Equal(t, []*model.Webhook{webhook1}, actualWebhooks)

		actualWebhooks, err = sqlStore.GetWebhooks(&model.WebhookFilter{Paging: model.Paging{Page: 0, PerPage: 10, IncludeDeleted: false}})
		require.NoError(t, err)
		require.Equal(t, []*model.Webhook{webhook1, webhook2}, actualWebhooks)

		actualWebhooks, err = sqlStore.GetWebhooks(&model.WebhookFilter{Paging: model.Paging{Page: 0, PerPage: 1, IncludeDeleted: true}})
		require.NoError(t, err)
		require.Equal(t, []*model.Webhook{webhook1}, actualWebhooks)

		actualWebhooks, err = sqlStore.GetWebhooks(&model.WebhookFilter{Paging: model.Paging{Page: 0, PerPage: 10, IncludeDeleted: true}})
		require.NoError(t, err)
		require.Equal(t, []*model.Webhook{webhook1, webhook2}, actualWebhooks)

		actualWebhooks, err = sqlStore.GetWebhooks(&model.WebhookFilter{Paging: model.AllPagesWithDeleted()})
		require.NoError(t, err)
		require.Equal(t, []*model.Webhook{webhook1, webhook2}, actualWebhooks)
	})

	t.Run("delete webhook", func(t *testing.T) {
		logger := testlib.MakeLogger(t)
		sqlStore := MakeTestSQLStore(t, logger)

		webhook1 := &model.Webhook{
			OwnerID: "owner1",
			URL:     "https://url1.com",
		}

		webhook2 := &model.Webhook{
			OwnerID: "owner2",
			URL:     "https://url2.com",
		}

		err := sqlStore.CreateWebhook(webhook1)
		require.NoError(t, err)

		time.Sleep(1 * time.Millisecond)

		err = sqlStore.CreateWebhook(webhook2)
		require.NoError(t, err)

		err = sqlStore.DeleteWebhook(webhook1.ID)
		require.NoError(t, err)

		actualWebhook1, err := sqlStore.GetWebhook(webhook1.ID)
		require.NoError(t, err)
		require.NotEqual(t, 0, actualWebhook1.DeleteAt)
		webhook1.DeleteAt = actualWebhook1.DeleteAt
		require.Equal(t, webhook1, actualWebhook1)

		actualWebhook2, err := sqlStore.GetWebhook(webhook2.ID)
		require.NoError(t, err)
		require.Equal(t, webhook2, actualWebhook2)

		actualWebhooks, err := sqlStore.GetWebhooks(&model.WebhookFilter{Paging: model.Paging{Page: 0, PerPage: 0, IncludeDeleted: false}})
		require.NoError(t, err)
		require.Empty(t, actualWebhooks)

		actualWebhooks, err = sqlStore.GetWebhooks(&model.WebhookFilter{Paging: model.Paging{Page: 0, PerPage: 1, IncludeDeleted: false}})
		require.NoError(t, err)
		require.Equal(t, []*model.Webhook{webhook2}, actualWebhooks)

		actualWebhooks, err = sqlStore.GetWebhooks(&model.WebhookFilter{Paging: model.Paging{Page: 0, PerPage: 10, IncludeDeleted: false}})
		require.NoError(t, err)
		require.Equal(t, []*model.Webhook{webhook2}, actualWebhooks)

		actualWebhooks, err = sqlStore.GetWebhooks(&model.WebhookFilter{Paging: model.Paging{Page: 0, PerPage: 1, IncludeDeleted: true}})
		require.NoError(t, err)
		require.Equal(t, []*model.Webhook{webhook1}, actualWebhooks)

		actualWebhooks, err = sqlStore.GetWebhooks(&model.WebhookFilter{Paging: model.Paging{Page: 0, PerPage: 10, IncludeDeleted: true}})
		require.NoError(t, err)
		require.Equal(t, []*model.Webhook{webhook1, webhook2}, actualWebhooks)

		time.Sleep(1 * time.Millisecond)

		// Deleting again shouldn't change timestamp
		err = sqlStore.DeleteWebhook(webhook1.ID)
		require.NoError(t, err)

		actualWebhook1, err = sqlStore.GetWebhook(webhook1.ID)
		require.NoError(t, err)
		require.Equal(t, webhook1, actualWebhook1)
	})
}
