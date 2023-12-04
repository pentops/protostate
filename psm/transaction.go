package psm

import (
	"context"

	"github.com/pentops/outbox.pg.go/outbox"
	"google.golang.org/protobuf/proto"
	"gopkg.daemonl.com/sqrlx"
)

type Transaction[State proto.Message, WrappedEvent proto.Message] interface {
	StoreEvent(context.Context, State, WrappedEvent) error
	Outbox(context.Context, outbox.OutboxMessage) error
}

type SqrlxTransaction[State proto.Message, WrappedEvent proto.Message] struct {
	sqrlx.Transaction
	callback func(context.Context, sqrlx.Transaction, State, WrappedEvent) error
}

func (st *SqrlxTransaction[State, WrappedEvent]) StoreEvent(ctx context.Context, state State, event WrappedEvent) error {

	return st.callback(ctx, st.Transaction, state, event)
}

func (st *SqrlxTransaction[State, WrappedEvent]) Outbox(ctx context.Context, msg outbox.OutboxMessage) error {

	return outbox.Send(ctx, st.Transaction, msg)
}

func NewSqrlxTransaction[State proto.Message, WrappedEvent proto.Message](
	tx sqrlx.Transaction,
	callback func(context.Context, sqrlx.Transaction, State, WrappedEvent) error,
) *SqrlxTransaction[State, WrappedEvent] {
	return &SqrlxTransaction[State, WrappedEvent]{
		Transaction: tx,
		callback:    callback,
	}
}
