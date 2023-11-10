package genericstate

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"gopkg.daemonl.com/sqrlx"

	sq "github.com/elgris/sqrl"
	"github.com/interxfi/go-api/event/v1/event_pb"
)

type StateSpec[E proto.Message, S proto.Message] struct {
	EmptyState func() S
	Transition func(ctx context.Context, state S, event E) error

	PrimaryKey        func(E) map[string]interface{}
	AdditionalColumns func(E) map[string]interface{}
	EventForeignKey   func(E) map[string]interface{}

	StateTable string
	EventTable string
}

type StateMachine[E proto.Message, S proto.Message] struct {
	db   *sqrlx.Wrapper
	spec StateSpec[E, S]
}

func NewStateMachine[E proto.Message, S proto.Message](db *sqrlx.Wrapper, spec StateSpec[E, S]) *StateMachine[E, S] {
	return &StateMachine[E, S]{
		db:   db,
		spec: spec,
	}
}

func (sm *StateMachine[E, S]) Transition(ctx context.Context, metadata *event_pb.Metadata, eventData E) error {

	stateSpec := sm.spec

	eventJSON, err := protojson.Marshal(eventData)
	if err != nil {
		return err
	}

	actorJSON, err := protojson.Marshal(metadata.Actor)
	if err != nil {
		return err
	}

	eventMap := map[string]interface{}{
		"id":        metadata.EventId,
		"timestamp": metadata.Timestamp.AsTime(),
		"event":     eventJSON,
		"actor":     actorJSON,
	}

	primaryKey := stateSpec.PrimaryKey(eventData)
	additionalColumns := stateSpec.AdditionalColumns(eventData)

	for k, v := range additionalColumns {
		eventMap[k] = v
	}

	selectQuery := sq.Select("state").From(stateSpec.StateTable)
	for k, v := range primaryKey {
		selectQuery = selectQuery.Where(sq.Eq{k: v})
	}

	for k, v := range stateSpec.EventForeignKey(eventData) {
		eventMap[k] = v
	}
	return sm.db.Transact(ctx, &sqrlx.TxOptions{
		Isolation: sql.LevelReadCommitted,
		ReadOnly:  false,
		Retryable: true,
	}, func(ctx context.Context, tx sqrlx.Transaction) error {
		state := stateSpec.EmptyState()
		err := tx.SelectRow(ctx, selectQuery).
			Scan(&state)
		if errors.Is(err, sql.ErrNoRows) {
		} else if err != nil {
			return fmt.Errorf("selecting current state: %w", err)
		}

		if err := stateSpec.Transition(ctx, state, eventData); err != nil {
			return fmt.Errorf("state transition: %w", err)
		}

		stateJSON, err := protojson.Marshal(state)
		if err != nil {
			return err
		}

		_, err = tx.Insert(ctx, sqrlx.Upsert(stateSpec.StateTable).
			KeyMap(primaryKey).
			SetMap(additionalColumns).
			Set("state", stateJSON))
		if err != nil {
			return fmt.Errorf("upsert state: %w", err)
		}

		_, err = tx.Insert(ctx, sq.Insert(stateSpec.EventTable).SetMap(eventMap))
		if err != nil {
			return fmt.Errorf("insert event: %w", err)
		}

		return nil
	})

}
