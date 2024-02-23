package psm

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	sq "github.com/elgris/sqrl"
	"github.com/pentops/log.go/log"
	"github.com/pentops/outbox.pg.go/outbox"
	"github.com/pentops/protostate/dbconvert"
	"github.com/pentops/sqrlx.go/sqrlx"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

type Transaction[State proto.Message, WrappedEvent proto.Message] interface {
	StoreEvent(context.Context, State, WrappedEvent) error
	Outbox(context.Context, outbox.OutboxMessage) error
	CheckEventExists(context.Context, WrappedEvent) (exists bool, err error)
}

type SqrlxTransaction[
	S IState[ST], // Outer State Entity
	ST IStatusEnum, // Status Enum in State Entity
	E IEvent[IE], // Event Wrapper, with IDs and Metadata
	IE IInnerEvent, // Inner Event, the typed event
] struct {
	tx   sqrlx.Transaction
	spec PSMTableSpec[S, ST, E, IE]
}

func NewSqrlxTransaction[
	S IState[ST], // Outer State Entity
	ST IStatusEnum, // Status Enum in State Entity
	E IEvent[IE], // Event Wrapper, with IDs and Metadata
	IE IInnerEvent, // Inner Event, the typed event
](
	tx sqrlx.Transaction,
	spec PSMTableSpec[S, ST, E, IE],
) *SqrlxTransaction[S, ST, E, IE] {
	return &SqrlxTransaction[S, ST, E, IE]{
		tx:   tx,
		spec: spec,
	}
}

func (st *SqrlxTransaction[S, ST, E, IE]) StoreEvent(ctx context.Context, state S, event E) error {
	var err error

	stateSpec := st.spec

	stateSetMap, err := stateSpec.State.storeDBMap(state)
	if err != nil {
		return fmt.Errorf("state fields: %w", err)
	}

	primaryKey, err := stateSpec.PrimaryKey(event)
	if err != nil {
		return fmt.Errorf("primary key: %w", err)
	}

	stateKeyMap, err := dbconvert.FieldsToDBValues(primaryKey)
	if err != nil {
		return fmt.Errorf("failed to map state primary key to DB values: %w", err)
	}

	_, err = st.tx.Insert(ctx, sqrlx.
		Upsert(stateSpec.State.TableName).
		KeyMap(stateKeyMap).
		SetMap(stateSetMap))

	if err != nil {
		log.WithFields(ctx, map[string]interface{}{
			"stateKeyMap": stateKeyMap,
			"stateSetMap": stateSetMap,
			"error":       err.Error(),
		}).Error("failed to upsert state")
		return fmt.Errorf("upsert state: %w", err)
	}

	eventSetMap, err := stateSpec.Event.storeDBMap(event)
	if err != nil {
		return fmt.Errorf("event fields: %w", err)
	}

	_, err = st.tx.Insert(ctx, sq.
		Insert(stateSpec.Event.TableName).
		SetMap(eventSetMap))
	if err != nil {
		return fmt.Errorf("insert event: %w", err)
	}

	return nil

}

func (st *SqrlxTransaction[S, ST, E, IE]) Outbox(ctx context.Context, msg outbox.OutboxMessage) error {
	return outbox.Send(ctx, st.tx, msg)
}

func (st *SqrlxTransaction[S, ST, E, IE]) CheckEventExists(ctx context.Context, event E) (exists bool, err error) {
	primaryKey, err := st.spec.Event.PK(event)
	if err != nil {
		return false, fmt.Errorf("primary key: %w", err)
	}
	pkMap, err := dbconvert.FieldsToDBValues(primaryKey)
	if err != nil {
		return false, fmt.Errorf("failed to map event primary key to DB values: %w", err)
	}

	selectQuery := sq.
		Select(st.spec.Event.DataColumn).
		From(st.spec.Event.TableName).
		Where(sq.Eq(pkMap))

	var data []byte
	err = st.tx.SelectRow(ctx, selectQuery).Scan(&data)
	if errors.Is(sql.ErrNoRows, err) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("selecting event: %w", err)
	}

	existing := (*new(E)).ProtoReflect().New()

	if err := protojson.Unmarshal(data, existing.Interface()); err != nil {
		return false, fmt.Errorf("unmarshalling event: %w", err)
	}

	if !proto.Equal(existing.Interface(), event) {
		return false, ErrDuplicateEventID
	}

	return true, nil
}

var ErrDuplicateEventID = errors.New("duplicate event ID")

var TxOptions = &sqrlx.TxOptions{
	Isolation: sql.LevelReadCommitted,
	Retryable: true,
	ReadOnly:  false,
}
