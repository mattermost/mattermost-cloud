// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package store

import (
	"database/sql"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
)

var (
	subscriptionsColumns = []string{
		"ID",
		"URL",
		"Name",
		"OwnerID",
		"EventType",
		"Headers",
		"FailureThreshold",
		"LastDeliveryStatus",
		"LastDeliveryAttemptAt",
		"CreateAt",
		"DeleteAt",
		"LockAcquiredBy",
		"LockAcquiredAt",
	}

	subscriptionsSelect = sq.Select(subscriptionsColumns...).
				From(subscriptionsTable)

	claimSubscriptionSelect = sq.Select(prefixAll("sub.", subscriptionsColumns)...).
				From(fmt.Sprintf("%s as sub", subscriptionsTable)).
				Join("EventDelivery ON sub.ID=EventDelivery.SubscriptionID").
				Where("DeleteAt = 0").
		// Take only not claimed subscriptions.
		Where("sub.LockAcquiredAt = 0").
		Where(sq.Eq{"LockAcquiredBy": nil}).
		// Start with subscriptions that were not processed recently.
		OrderBy("sub.LastDeliveryAttemptAt ASC").
		Limit(1)
)

// ErrNoSubscriptionsToProcess indicates that there is no subscription to claim.
var ErrNoSubscriptionsToProcess error = fmt.Errorf("no subscriptions to process")

// ClaimUpToDateSubscription fetches and locks first subscription which last delivery did not fail and has events to process.
func (sqlStore *SQLStore) ClaimUpToDateSubscription(instanceID string) (*model.Subscription, error) {
	query := claimSubscriptionSelect.
		// Only new and succeeded on last attempt.
		Where(sq.Eq{"sub.LastDeliveryStatus": []model.SubscriptionDeliveryStatus{model.SubscriptionDeliverySucceeded, model.SubscriptionDeliveryNone}}).
		// Have new, not delivered events.
		Where("EventDelivery.status = ?", model.EventDeliveryNotAttempted)

	return sqlStore.claimSubscription(instanceID, query)
}

// ClaimRetryingSubscription fetches and locks first subscription which last delivery failed and has events to process.
func (sqlStore *SQLStore) ClaimRetryingSubscription(instanceID string, cooldown time.Duration) (*model.Subscription, error) {
	minTime := model.GetMillis() - cooldown.Milliseconds()

	query := claimSubscriptionSelect.
		// Failed on last delivery.
		Where("sub.LastDeliveryStatus = ?", model.SubscriptionDeliveryFailed).
		// More than some time ago.
		Where("sub.LastDeliveryAttemptAt < ?", minTime).
		// And have retrying or new events.
		Where(sq.Eq{"EventDelivery.status": []model.EventDeliveryStatus{model.EventDeliveryRetrying, model.EventDeliveryNotAttempted}})

	return sqlStore.claimSubscription(instanceID, query)
}

func (sqlStore *SQLStore) claimSubscription(instanceID string, query sq.SelectBuilder) (*model.Subscription, error) {
	tx, err := sqlStore.beginTransaction(sqlStore.db)
	if err != nil {
		return nil, errors.Wrap(err, "failed to begin transaction")
	}
	defer tx.RollbackUnlessCommitted()

	if sqlStore.db.DriverName() == driverPostgres {
		// To avoid conflicts on custom lock, we make Postgres lock the row
		// for the time of transaction with `FOR UPDATE`.
		// For multiple calls to not block when asking for the same row,
		// we use `SKIP LOCKED` as we only need one row that matches our expectations.
		query = query.Suffix("FOR UPDATE SKIP LOCKED")
	}

	subscriptions := []*model.Subscription{}
	err = sqlStore.selectBuilder(tx, &subscriptions, query)
	if err != nil {
		return nil, errors.Wrap(err, "failed to claim subscription")
	}

	if len(subscriptions) == 0 {
		return nil, ErrNoSubscriptionsToProcess
	}
	if len(subscriptions) > 1 {
		return nil, errors.Errorf("expected only one subscription")
	}

	sub := subscriptions[0]

	locked, err := sqlStore.lockSubscription(tx, sub.ID, instanceID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to lock subscription")
	}
	if !locked {
		return nil, errors.New("failed to lock subscription")
	}

	err = tx.Commit()
	if err != nil {
		return nil, errors.Wrap(err, "failed to commit transaction")
	}
	return sub, nil
}

// CountSubscriptionsForEvent count number of subscriptions for the specified event type.
func (sqlStore *SQLStore) CountSubscriptionsForEvent(eventType model.EventType) (int64, error) {
	query := sq.Select("Count (*)").From(subscriptionsTable).
		Where("eventType = ?", eventType).
		Where("DeleteAt = 0")

	count, err := sqlStore.getCount(query)
	if err != nil {
		return 0, errors.Wrap(err, "failed to get subscriptions count")
	}
	return count, nil
}

// CreateSubscription creates new subscription.
func (sqlStore *SQLStore) CreateSubscription(sub *model.Subscription) error {
	sub.ID = model.NewID()
	sub.CreateAt = model.GetMillis()

	_, err := sqlStore.execBuilder(sqlStore.db, sq.Insert(subscriptionsTable).
		SetMap(map[string]interface{}{
			"ID":                    sub.ID,
			"Name":                  sub.Name,
			"URL":                   sub.URL,
			"OwnerID":               sub.OwnerID,
			"EventType":             sub.EventType,
			"LastDeliveryAttemptAt": sub.LastDeliveryAttemptAt,
			"LastDeliveryStatus":    sub.LastDeliveryStatus,
			"FailureThreshold":      sub.FailureThreshold,
			"CreateAt":              sub.CreateAt,
			"DeleteAt":              sub.DeleteAt,
			"LockAcquiredAt":        sub.LockAcquiredAt,
			"LockAcquiredBy":        sub.LockAcquiredBy,
			"Headers":               sub.Headers,
		}))
	if err != nil {
		return errors.Wrap(err, "failed to create subscription")
	}

	return nil
}

// UpdateSubscriptionStatus updates subscription status.
func (sqlStore *SQLStore) UpdateSubscriptionStatus(sub *model.Subscription) error {
	_, err := sqlStore.execBuilder(sqlStore.db, sq.Update(subscriptionsTable).
		SetMap(map[string]interface{}{
			"LastDeliveryStatus":    sub.LastDeliveryStatus,
			"LastDeliveryAttemptAt": sub.LastDeliveryAttemptAt,
		}).
		Where("ID = ?", sub.ID).
		Where("DeleteAt = 0"),
	)
	if err != nil {
		return errors.Wrap(err, "failed to update event delivery status")
	}

	return nil
}

// GetSubscriptions fetches subscriptions specified by the filter.
func (sqlStore *SQLStore) GetSubscriptions(filter *model.SubscriptionsFilter) ([]*model.Subscription, error) {
	return sqlStore.getSubscriptions(sqlStore.db, filter)
}

func (sqlStore *SQLStore) getSubscriptions(db queryer, filter *model.SubscriptionsFilter) ([]*model.Subscription, error) {
	query := subscriptionsSelect.
		OrderBy("CreateAt DESC")
	query = applyPagingFilter(query, filter.Paging)

	if filter.EventType != "" {
		query = query.Where("eventType = ?", filter.EventType)
	}
	if filter.Owner != "" {
		query = query.Where("ownerID = ?", filter.Owner)
	}

	subs := []*model.Subscription{}
	err := sqlStore.selectBuilder(db, &subs, query)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get subscription")
	}

	return subs, nil
}

// GetSubscription fetches a subscription by ID.
func (sqlStore *SQLStore) GetSubscription(subID string) (*model.Subscription, error) {
	sub := model.Subscription{}
	err := sqlStore.getBuilder(sqlStore.db, &sub, subscriptionsSelect.Where("ID = ?", subID))
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, errors.Wrap(err, "failed to get subscription")
	}

	return &sub, nil
}

// DeleteSubscription marks the given subscription as deleted.
func (sqlStore *SQLStore) DeleteSubscription(id string) error {
	_, err := sqlStore.execBuilder(sqlStore.db, sq.
		Update(subscriptionsTable).
		Set("DeleteAt", model.GetMillis()).
		Where("ID = ?", id).
		Where("DeleteAt = 0"),
	)
	if err != nil {
		return errors.Wrap(err, "failed to mark subscription as deleted")
	}

	return nil
}

func prefixAll(prefix string, strs []string) []string {
	out := make([]string, len(strs))
	for i := range strs {
		out[i] = fmt.Sprintf("%s%s", prefix, strs[i])
	}
	return out
}
