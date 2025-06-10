package psm

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	sq "github.com/elgris/sqrl"
	"github.com/pentops/log.go/log"
	"github.com/pentops/protostate/internal/dbconvert"
	"github.com/pentops/sqrlx.go/sqrlx"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// causeEventIdempotency checks if the event ID for the outer triggering event
// is unique in the event table. If not, it checks if the event is a repeat
// processing of the same event, and returns the state after the initial
// transition.
func (sm *StateMachine[K, S, ST, SD, E, IE]) causeEventIdempotency(ctx context.Context, tx sqrlx.Transaction, se preparedEvent[K, S, ST, SD, E, IE]) (S, bool, error) {
	var s S
	selectQuery := sq.
		Select(sm.tableMap.Event.Root.ColumnName).
		From(sm.tableMap.Event.TableName).
		Where(sq.Eq{sm.tableMap.Event.IdempotencyHash.ColumnName: se.idempotencyKey})

	selectQuery.Column(sm.tableMap.Event.StateSnapshot.ColumnName)

	var eventData, stateData []byte
	err := tx.SelectRow(ctx, selectQuery).Scan(&eventData, &stateData)
	if errors.Is(err, sql.ErrNoRows) {
		return s, false, nil
	}
	if err != nil {
		return s, false, fmt.Errorf("selecting event: %w", err)
	}

	existing := (*new(E)).ProtoReflect().New().Interface().(E)

	if err := protojson.Unmarshal(eventData, existing); err != nil {
		return s, false, fmt.Errorf("unmarshalling event: %w", err)
	}

	if !proto.Equal(existing.UnwrapPSMEvent(), se.event.UnwrapPSMEvent()) {
		return s, false, ErrDuplicateEventID
	}

	state := (*new(S)).ProtoReflect().New()
	if err := protojson.Unmarshal(stateData, state.Interface()); err != nil {
		return s, false, fmt.Errorf("unmarshalling state: %w", err)
	}

	return state.Interface().(S), true, nil
}

// followEventDeduplicate is similar to the firstEventUniqueCheck, but it
// compares the entire event including metadata, as this is not designed to
// handle consumer idempotency.
func (sm *StateMachine[K, S, ST, SD, E, IE]) followEventDeduplicate(ctx context.Context, tx sqrlx.Transaction, se preparedEvent[K, S, ST, SD, E, IE]) (bool, error) {
	selectQuery := sq.
		Select(sm.tableMap.Event.Root.ColumnName).
		From(sm.tableMap.Event.TableName).
		Where(sq.Eq{sm.tableMap.Event.ID.ColumnName: se.event.PSMMetadata().EventId})

	var eventData, stateData []byte
	err := tx.SelectRow(ctx, selectQuery).Scan(&eventData, &stateData)

	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("selecting event for deduplication: %w", err)
	}

	existing := (*new(E)).ProtoReflect().New().Interface().(E)
	if err := protojson.Unmarshal(eventData, existing); err != nil {
		return true, fmt.Errorf("unmarshalling event: %w", err)
	}

	if !proto.Equal(existing, se.event) {
		return true, fmt.Errorf("event %s already exists with different data", existing.PSMMetadata().EventId)
	}

	return true, nil
}

func (sm *StateMachine[K, S, ST, SD, E, IE]) storeStateAndEvent(
	ctx context.Context,
	tx sqrlx.Transaction,
	evt preparedEvent[K, S, ST, SD, E, IE],
) error {
	state := evt.state
	event := evt.event

	if state.GetStatus() == 0 {
		return fmt.Errorf("state machine transitioned to zero status")
	}

	err := sm.validator.Validate(state)
	if err != nil {
		return err
	}

	stateDBValue, err := dbconvert.MarshalProto(state)
	if err != nil {
		return fmt.Errorf("state field: %w", err)
	}

	eventDBValue, err := dbconvert.MarshalProto(event)
	if err != nil {
		return fmt.Errorf("event field: %w", err)
	}

	// TODO: This does not change during transitions, so should be calculated
	// early and once.
	keyValues, err := sm.keyValues(state.PSMKeys())
	if err != nil {
		return fmt.Errorf("key fields: %w", err)
	}
	if len(keyValues.missingRequired) > 0 {
		return fmt.Errorf("missing required key(s) %v in store", keyValues.missingRequired)
	}

	eventMeta := event.PSMMetadata()

	upsertStateQuery := sqrlx.Upsert(sm.tableMap.State.TableName)

	insertValues := []any{}
	insertColumns := []string{}

	insertEventQuery := sq.Insert(sm.tableMap.Event.TableName)

	insertColumns = append(insertColumns, sm.tableMap.Event.ID.ColumnName)
	insertValues = append(insertValues, eventMeta.EventId)

	for _, key := range keyValues.values {
		if key.Primary {
			upsertStateQuery.Key(key.ColumnName, key.value)
		} else {
			upsertStateQuery.Set(key.ColumnName, key.value)
		}

		insertColumns = append(insertColumns, key.ColumnName)
		insertValues = append(insertValues, key.value)
	}

	insertColumns = append(insertColumns,
		sm.tableMap.Event.IdempotencyHash.ColumnName,
		sm.tableMap.Event.Timestamp.ColumnName,
		sm.tableMap.Event.Sequence.ColumnName,
		sm.tableMap.Event.Root.ColumnName,
		sm.tableMap.Event.StateSnapshot.ColumnName,
	)
	insertValues = append(insertValues,
		evt.idempotencyKey,
		eventMeta.Timestamp.AsTime(),
		eventMeta.Sequence,
		eventDBValue,
		stateDBValue,
	)
	insertEventQuery.Columns(insertColumns...).Values(insertValues...)

	upsertStateQuery.Set(sm.tableMap.State.Root.ColumnName, stateDBValue)

	_, err = tx.Insert(ctx, upsertStateQuery)
	if err != nil {
		log.WithFields(ctx, map[string]any{
			"keys":  keyValues,
			"error": err.Error(),
		}).Error("failed to upsert state")
		return fmt.Errorf("upsert state: %w", err)
	}

	_, err = tx.Insert(ctx, insertEventQuery)
	if err != nil {
		log.WithFields(ctx, map[string]any{
			"keys":  keyValues,
			"error": err.Error(),
		}).Error("failed to insert event")
		return fmt.Errorf("insert event: %w", err)
	}

	return nil
}

func (sm *StateMachine[K, S, ST, SD, E, IE]) getCurrentState(ctx context.Context, tx sqrlx.Transaction, keys K) (S, error) {
	state := (*new(S)).ProtoReflect().New().Interface().(S)

	selectQuery := sq.
		Select(sm.tableMap.State.Root.ColumnName).
		From(sm.tableMap.State.TableName)

	allKeys, err := sm.keyValues(keys)
	if err != nil {
		return state, err
	}
	for _, key := range allKeys.values {
		if !key.Primary {
			continue
		}
		selectQuery = selectQuery.Where(sq.Eq{key.ColumnName: key.value})
	}

	var stateJSON []byte
	err = tx.SelectRow(ctx, selectQuery).Scan(&stateJSON)
	if errors.Is(err, sql.ErrNoRows) {
		state.SetPSMKeys(proto.Clone(keys).(K))

		if len(allKeys.missingRequired) > 0 {
			return state, fmt.Errorf("missing required key(s) %v in initial event", allKeys.missingRequired)
		}

		// OK, leave empty state alone
		return state, nil
	}
	if err != nil {
		qq, _, _ := selectQuery.ToSql()
		return state, fmt.Errorf("selecting current state (%s): %w", qq, err)
	}

	if err := protojson.Unmarshal(stateJSON, state); err != nil {
		return state, err
	}

	return state, nil
}
