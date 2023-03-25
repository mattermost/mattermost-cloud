// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package store

import (
	"database/sql"

	sq "github.com/Masterminds/squirrel"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
)

var webhookSelect sq.SelectBuilder

func init() {
	webhookSelect = sq.
		Select("ID", "OwnerID", "URL", "Headers", "CreateAt", "DeleteAt").From("Webhooks")
}

// GetWebhook fetches the given webhook by id.
func (sqlStore *SQLStore) GetWebhook(id string) (*model.Webhook, error) {
	var webhook model.Webhook
	err := sqlStore.getBuilder(sqlStore.db, &webhook,
		webhookSelect.Where("ID = ?", id),
	)
	if err == sql.ErrNoRows {
		return nil, nil
	} else if err != nil {
		return nil, errors.Wrap(err, "failed to get webhook by id")
	}

	return &webhook, nil
}

// GetWebhooks fetches the given page of created webhooks. The first page is 0.
func (sqlStore *SQLStore) GetWebhooks(filter *model.WebhookFilter) ([]*model.Webhook, error) {
	builder := webhookSelect.
		OrderBy("CreateAt ASC")

	builder = applyPagingFilter(builder, filter.Paging)

	if filter.OwnerID != "" {
		builder = builder.Where("OwnerID = ?", filter.OwnerID)
	}

	var webhooks []*model.Webhook
	err := sqlStore.selectBuilder(sqlStore.db, &webhooks, builder)
	if err != nil {
		return nil, errors.Wrap(err, "failed to query for webhooks")
	}

	return webhooks, nil
}

// CreateWebhook records the given webhook to the database, assigning it a unique ID.
func (sqlStore *SQLStore) CreateWebhook(webhook *model.Webhook) error {
	webhook.ID = model.NewID()
	webhook.CreateAt = model.GetMillis()

	_, err := sqlStore.execBuilder(sqlStore.db, sq.
		Insert("Webhooks").
		SetMap(map[string]interface{}{
			"ID":       webhook.ID,
			"OwnerID":  webhook.OwnerID,
			"URL":      webhook.URL,
			"CreateAt": webhook.CreateAt,
			"DeleteAt": 0,
			"Headers":  webhook.Headers,
		}),
	)
	if err != nil {
		return errors.Wrap(err, "failed to create webhook")
	}

	return nil
}

// DeleteWebhook marks the given webhook as deleted, but does not remove the
// record from the database.
func (sqlStore *SQLStore) DeleteWebhook(id string) error {
	_, err := sqlStore.execBuilder(sqlStore.db, sq.
		Update("Webhooks").
		Set("DeleteAt", model.GetMillis()).
		Where("ID = ?", id).
		Where("DeleteAt = 0"),
	)
	if err != nil {
		return errors.Wrap(err, "failed to mark webhook as deleted")
	}

	return nil
}
