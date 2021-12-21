// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package testutil

import (
	"context"

	"github.com/mattermost/mattermost-cloud/internal/events"
	"github.com/mattermost/mattermost-cloud/internal/store"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/sirupsen/logrus"
)

// SetupTestEventsProducer sets up testing event produced that does not deliver events.
func SetupTestEventsProducer(sqlStore *store.SQLStore, logger logrus.FieldLogger) *events.EventProducer {
	cfg := events.DelivererConfig{
		RetryWorkers:    0,
		UpToDateWorkers: 0,
		MaxBurstWorkers: 0,
	}
	deliverer := events.NewDeliverer(context.Background(), sqlStore, model.NewID(), logger, cfg)

	return events.NewProducer(sqlStore, deliverer, "test", logger)
}
