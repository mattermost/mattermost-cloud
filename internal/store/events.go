// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package store

import (
	"database/sql"
	"encoding/json"

	sq "github.com/Masterminds/squirrel"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
)

const (
	subscriptionsTable    = "Subscription"
	eventTable            = "Event"
	eventDeliveryTable    = "EventDelivery"
	stateChangeEventTable = "StateChangeEvent"
)

var (
	eventDeliveryColumns = []string{"ID", "EventID", "SubscriptionID", "Status", "LastAttempt", "Attempts"}

	stateChangeEventSelect = sq.Select("sc.ID, sc.ResourceID, sc.ResourceType, sc.OldState, sc.NewState, sc.EventID, e.Timestamp, e.EventType, e.ExtraData").
				From("StateChangeEvent as sc").
				Join("Event as e on sc.EventID = e.ID")
)

// CreateStateChangeEvent creates new StateChangeEvent and initializes EventDeliveries.
func (sqlStore *SQLStore) CreateStateChangeEvent(event *model.StateChangeEventData) error {
	tx, err := sqlStore.beginTransaction(sqlStore.db)
	if err != nil {
		return errors.Wrap(err, "failed to begin transaction")
	}
	defer tx.RollbackUnlessCommitted()

	err = sqlStore.createEvent(tx, &event.Event)
	if err != nil {
		return errors.Wrap(err, "failed to create event")
	}

	event.StateChange.EventID = event.Event.ID

	err = sqlStore.createStateChangeEvent(tx, &event.StateChange)
	if err != nil {
		return errors.Wrap(err, "failed to create state change event")
	}

	subFilter := model.SubscriptionsFilter{EventType: event.Event.EventType, Paging: model.AllPagesNotDeleted()}
	subscriptions, err := sqlStore.getSubscriptions(tx, &subFilter)
	if err != nil {
		return errors.Wrap(err, "failed to get subscriptions")
	}

	err = sqlStore.createEventDeliveries(tx, &event.Event, subscriptions)
	if err != nil {
		return errors.Wrap(err, "failed to create event deliveries")
	}

	err = tx.Commit()
	if err != nil {
		return errors.Wrap(err, "failed to commit transaction")
	}
	return nil
}

func (sqlStore *SQLStore) createEvent(db execer, event *model.Event) error {
	event.ID = model.NewID()

	extraData, err := json.Marshal(event.ExtraData)
	if err != nil {
		return errors.Wrap(err, "failed to marshal events' extra data")
	}

	_, err = sqlStore.execBuilder(db, sq.
		Insert(eventTable).
		SetMap(map[string]interface{}{
			"ID":        event.ID,
			"EventType": event.EventType,
			"Timestamp": event.Timestamp,
			"ExtraData": extraData,
		}),
	)
	if err != nil {
		return errors.Wrap(err, "failed to create event")
	}
	return nil
}

func (sqlStore *SQLStore) createStateChangeEvent(db execer, event *model.StateChangeEvent) error {
	event.ID = model.NewID()

	_, err := sqlStore.execBuilder(db, sq.
		Insert(stateChangeEventTable).
		SetMap(map[string]interface{}{
			"ID":           event.ID,
			"EventID":      event.EventID,
			"ResourceID":   event.ResourceID,
			"ResourceType": event.ResourceType,
			"OldState":     event.OldState,
			"NewState":     event.NewState,
		}),
	)
	if err != nil {
		return errors.Wrap(err, "failed to create state change event")
	}
	return nil
}

func (sqlStore *SQLStore) createEventDeliveries(db dbInterface, event *model.Event, subscriptions []*model.Subscription) error {
	if len(subscriptions) == 0 {
		return nil
	}

	// Although we do not expect huge number of subscriptions
	// max number of prepared statement tokens is 999, so we batch
	// for sake of future proofing.
	batchStart := 0
	batchSize := 50
	for batchStart < len(subscriptions) {
		end := batchStart + batchSize
		if end > len(subscriptions) {
			end = len(subscriptions)
		}
		err := sqlStore.insertEventDeliveries(db, event, subscriptions[batchStart:end])
		if err != nil {
			return err
		}
		batchStart += batchSize
	}

	return nil
}

func (sqlStore *SQLStore) insertEventDeliveries(db dbInterface, event *model.Event, subscriptions []*model.Subscription) error {
	builder := sq.Insert("EventDelivery").Columns(eventDeliveryColumns...)
	for _, sub := range subscriptions {
		builder = builder.Values(model.NewID(), event.ID, sub.ID, model.EventDeliveryNotAttempted, 0, 0)
	}

	_, err := sqlStore.execBuilder(db, builder)
	if err != nil {
		return errors.Wrap(err, "failed to create event deliveries")
	}
	return nil
}

// stateChangeEventData is a helper struct for querying joined data of Event and StateChangeEvent.
type stateChangeEventData struct {
	EventType model.EventType
	Timestamp int64
	ExtraData []byte
	model.StateChangeEvent
}

func (s stateChangeEventData) toStateChangeEventData() (model.StateChangeEventData, error) {
	extraData := model.EventExtraData{}
	err := json.Unmarshal(s.ExtraData, &extraData)
	if err != nil {
		return model.StateChangeEventData{}, errors.Wrapf(err, "failed to unmarshal events' extra data, eventID: %q", s.EventID)
	}

	return model.StateChangeEventData{
		Event: model.Event{
			ID:        s.EventID,
			EventType: s.EventType,
			Timestamp: s.Timestamp,
			ExtraData: extraData,
		},
		StateChange: s.StateChangeEvent,
	}, nil
}

// GetStateChangeEventsToProcess returns StateChangeEventDeliveryData for given subscription in order of occurrence.
func (sqlStore *SQLStore) GetStateChangeEventsToProcess(subID string) ([]*model.StateChangeEventDeliveryData, error) {
	var eventDeliveries []*model.EventDelivery
	err := sqlStore.selectBuilder(sqlStore.db, &eventDeliveries,
		sq.Select(eventDeliveryColumns...).
			From(eventDeliveryTable).
			Where("SubscriptionID = ?", subID).
			Where(sq.Eq{"Status": []model.EventDeliveryStatus{model.EventDeliveryNotAttempted, model.EventDeliveryRetrying}}),
	)
	if err != nil {
		return nil, errors.Wrap(err, "failed to query event deliveries for subscription")
	}

	eventIDs := make([]string, 0, len(eventDeliveries))
	for _, e := range eventDeliveries {
		eventIDs = append(eventIDs, e.EventID)
	}

	var eventsData []stateChangeEventData
	err = sqlStore.selectBuilder(sqlStore.db, &eventsData,
		stateChangeEventSelect.
			Where(sq.Eq{"sc.EventID": eventIDs}).
			OrderBy("e.Timestamp ASC"),
	)
	if err != nil {
		return nil, errors.Wrap(err, "failed to query state change events for subscription")
	}

	if len(eventsData) != len(eventDeliveries) {
		return nil, errors.Errorf("number of found events does not match number of deliveries, events: %d, deliveries: %d",
			len(eventsData), len(eventDeliveries))
	}

	deliveryData := make([]*model.StateChangeEventDeliveryData, len(eventsData))
	for i, event := range eventsData {
		delivery, found := model.EventDeliveryForEvent(event.EventID, eventDeliveries)
		if !found {
			return nil, errors.Wrap(err, "failed to find event delivery for the event")
		}

		eventData, err := event.toStateChangeEventData()
		if err != nil {
			return nil, err
		}

		deliveryData[i] = &model.StateChangeEventDeliveryData{
			EventDelivery: *delivery,
			EventData:     eventData,
		}
	}

	return deliveryData, nil
}

// GetStateChangeEvent fetches StateChangeEventData based on specified event ID.
func (sqlStore *SQLStore) GetStateChangeEvent(eventID string) (*model.StateChangeEventData, error) {
	var event stateChangeEventData
	err := sqlStore.getBuilder(sqlStore.db, &event,
		stateChangeEventSelect.Where("e.ID = ?", eventID),
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, errors.Wrap(err, "failed to get event")
	}

	eventData, err := event.toStateChangeEventData()
	if err != nil {
		return nil, err
	}
	return &eventData, nil
}

// GetStateChangeEvents fetches StateChangeEventData based on the filter.
func (sqlStore *SQLStore) GetStateChangeEvents(filter *model.StateChangeEventFilter) ([]*model.StateChangeEventData, error) {
	query := stateChangeEventSelect.OrderBy("e.Timestamp DESC")

	if filter.Paging.PerPage != model.AllPerPage {
		query = query.
			Limit(uint64(filter.Paging.PerPage)).
			Offset(uint64(filter.Paging.Page * filter.Paging.PerPage))
	}

	if filter.ResourceType != "" {
		query = query.Where("sc.ResourceType = ?", filter.ResourceType)
	}
	if filter.ResourceID != "" {
		query = query.Where("sc.ResourceID = ?", filter.ResourceID)
	}

	var eventsData []stateChangeEventData
	err := sqlStore.selectBuilder(sqlStore.db, &eventsData, query)
	if err != nil {
		return nil, errors.Wrap(err, "failed to query state change events")
	}

	out := make([]*model.StateChangeEventData, len(eventsData))
	for i, ed := range eventsData {
		data, err := ed.toStateChangeEventData()
		if err != nil {
			return nil, err
		}

		out[i] = &data
	}

	return out, nil
}

// UpdateEventDeliveryStatus updates status fields of EventDelivery.
func (sqlStore *SQLStore) UpdateEventDeliveryStatus(delivery *model.EventDelivery) error {
	_, err := sqlStore.execBuilder(sqlStore.db, sq.Update(eventDeliveryTable).
		SetMap(map[string]interface{}{
			"Status":      delivery.Status,
			"Attempts":    delivery.Attempts,
			"LastAttempt": delivery.LastAttempt,
		}).
		Where("ID = ?", delivery.ID),
	)
	if err != nil {
		return errors.Wrap(err, "failed to update event delivery status")
	}

	return nil
}

// GetDeliveriesForSubscription is a helper function used for some tests.
func (sqlStore *SQLStore) GetDeliveriesForSubscription(subID string) ([]*model.EventDelivery, error) {
	query := sq.Select("*").From(eventDeliveryTable).
		Where("SubscriptionID = ?", subID)

	deliveries := []*model.EventDelivery{}
	err := sqlStore.selectBuilder(sqlStore.db, &deliveries, query)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get event delivery")
	}

	return deliveries, nil
}

// lockSubscription marks the subscription as locked for exclusive use by the caller.
func (sqlStore *SQLStore) lockSubscription(db execer, subID, lockerID string) (bool, error) {
	return sqlStore.lockRowsTx(db, subscriptionsTable, []string{subID}, lockerID)
}

// UnlockSubscription releases a lock previously acquired against a caller.
func (sqlStore *SQLStore) UnlockSubscription(subID, lockerID string, force bool) (bool, error) {
	return sqlStore.unlockRows(subscriptionsTable, []string{subID}, lockerID, force)
}
