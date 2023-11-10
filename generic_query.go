package genericstate

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"buf.build/gen/go/bufbuild/protovalidate/protocolbuffers/go/buf/validate"
	sq "github.com/elgris/sqrl"
	"github.com/lib/pq"
	"github.com/pentops/log.go/log"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
	"gopkg.daemonl.com/sqrlx"
)

type Transactor interface {
	Transact(ctx context.Context, opts *sqrlx.TxOptions, callback sqrlx.Callback) error
}

type AuthProvider interface {
	AuthFilter(ctx context.Context) (map[string]interface{}, error)
}

type AuthProviderFunc func(ctx context.Context) (map[string]interface{}, error)

func (f AuthProviderFunc) AuthFilter(ctx context.Context) (map[string]interface{}, error) {
	return f(ctx)
}

type GenericQuery struct {
	StateTableName string
	StateColumn    string

	PrimaryKeyColumn string
	PrimaryKeyField  protoreflect.Name

	EventTableName        string
	EventForeignKeyColumn string
	EventSequenceColumn   string
	EventDataColumn       string

	DB Transactor
}

func (gc GenericQuery) validate() error {
	if gc.StateTableName == "" {
		return fmt.Errorf("missing StateTableName")
	}
	if gc.StateColumn == "" {
		return fmt.Errorf("missing StateColumn")
	}
	if gc.PrimaryKeyColumn == "" {
		return fmt.Errorf("missing PrimaryKeyColumn")
	}
	if gc.PrimaryKeyField == "" {
		return fmt.Errorf("missing PrimaryKeyField")
	}
	if gc.EventTableName == "" {
		return fmt.Errorf("missing EventTableName")
	}
	if gc.EventForeignKeyColumn == "" {
		return fmt.Errorf("missing EventForeignKeyColumn")
	}
	if gc.EventSequenceColumn == "" {
		return fmt.Errorf("missing EventSequenceColumn")
	}
	if gc.EventDataColumn == "" {
		return fmt.Errorf("missing EventDataColumn")
	}
	if gc.DB == nil {
		return fmt.Errorf("missing DB")
	}

	return nil
}

const (
	StateAlias = "_gc_state"
	EventAlias = "_gc_event"
)

func (gc *GenericQuery) Get(ctx context.Context, reqMsg proto.Message, resMsg proto.Message, constraintFields map[string]interface{}) error {
	if err := gc.validate(); err != nil {
		return err
	}

	resReflect := resMsg.ProtoReflect()
	reqReflect := reqMsg.ProtoReflect()

	var eventField protoreflect.FieldDescriptor     // wraps the ListFooEventResponse type
	var eventListField protoreflect.FieldDescriptor // the events array inside the response
	var stateField protoreflect.FieldDescriptor
	var requestPKField protoreflect.FieldDescriptor

	{ // TODO: Load on construction of GenericQuery

		reqDesc := reqReflect.Descriptor()

		// TODO: Use an annotation not a passed in name
		requestPKField = reqDesc.Fields().ByName(gc.PrimaryKeyField)
		if requestPKField == nil {
			return fmt.Errorf("request message has no field %s: %s", gc.PrimaryKeyField, reqDesc.FullName())
		}

		resDesc := resReflect.Descriptor()
		// TODO: Run this loop off annotations
		for i := 0; i < resDesc.Fields().Len(); i++ {
			field := resDesc.Fields().Get(i)
			fieldName := field.Name()
			if fieldName == "events" {
				eventField = field
				continue
			}
			if stateField != nil {
				return fmt.Errorf("multiple state fields (%s, %s)", stateField.Name(), field.Name())
			}
			stateField = field
		}

		if stateField == nil {
			return fmt.Errorf("no state field")
		}

		if eventField != nil {
			for i := 0; i < eventField.Message().Fields().Len(); i++ {
				field := eventField.Message().Fields().Get(i)
				fieldName := field.Name()
				if fieldName == "events" {
					eventListField = field
					break
				}
			}

			if eventListField == nil {
				return fmt.Errorf("no event list field")
			}
		}
	}

	idVal := reqReflect.Get(requestPKField).Interface()

	selectQuery := sq.
		Select().
		Column(fmt.Sprintf("%s.%s AS stateCol", StateAlias, gc.StateColumn)).
		From(fmt.Sprintf("%s AS %s", gc.StateTableName, StateAlias)).
		Where(sq.Eq{
			fmt.Sprintf("%s.%s", StateAlias, gc.PrimaryKeyColumn): idVal,
		}).GroupBy(fmt.Sprintf("%s.%s", StateAlias, gc.PrimaryKeyColumn))

	if eventField != nil {
		selectQuery.
			Column(fmt.Sprintf("ARRAY_AGG(%s.%s) AS eventsCol", EventAlias, gc.EventDataColumn)).
			LeftJoin(fmt.Sprintf(
				"%s AS %s ON %s.%s = %s.%s",
				gc.EventTableName,
				EventAlias,
				EventAlias,
				gc.EventForeignKeyColumn,
				StateAlias,
				gc.PrimaryKeyColumn,
			))
	}

	var foundJSON []byte
	var eventJSON pq.ByteaArray

	if err := gc.DB.Transact(ctx, &sqrlx.TxOptions{
		ReadOnly:  true,
		Retryable: true,
		Isolation: sql.LevelReadCommitted,
	}, func(ctx context.Context, tx sqrlx.Transaction) error {
		row := tx.SelectRow(ctx, selectQuery)

		var err error
		if eventField != nil {
			err = row.Scan(&foundJSON, &eventJSON)
		} else {
			err = row.Scan(&foundJSON)
		}
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return status.Errorf(codes.NotFound, "entity %s not found", idVal)
			}
			return err
		}

		return nil
	}); err != nil {
		return err
	}

	if foundJSON == nil {
		return status.Error(codes.NotFound, "not found")
	}

	stateMsg := resReflect.NewField(stateField)
	if err := protojson.Unmarshal(foundJSON, stateMsg.Message().Interface()); err != nil {
		return err
	}
	resReflect.Set(stateField, stateMsg)

	if eventField != nil {
		eventRes := resReflect.NewField(eventField).Message()
		eventList := eventRes.Mutable(eventListField).List()
		for _, eventBytes := range eventJSON {
			rowMessage := eventList.NewElement().Message()
			if err := protojson.Unmarshal(eventBytes, rowMessage.Interface()); err != nil {
				return err
			}
			eventList.Append(protoreflect.ValueOf(rowMessage))
		}

		resReflect.Set(eventField, protoreflect.ValueOf(eventRes)) // TODO: This may not be required
	}

	return nil

}

func (gc *GenericQuery) List(ctx context.Context, reqMsg proto.Message, resMsg proto.Message, constrain func(*sq.SelectBuilder)) error {

	var pageSize = uint64(20)
	var nextTokenField protoreflect.FieldDescriptor
	var arrayField protoreflect.FieldDescriptor
	res := resMsg.ProtoReflect()

	{ // Prepare
		// TODO: Build this on construction of GenericQuery

		resDesc := res.Descriptor()
		fields := resDesc.Fields()

		for i := 0; i < fields.Len(); i++ {
			field := fields.Get(i)
			if field.Name() == "next_token" {
				nextTokenField = field
				continue
			}
			if field.Cardinality() == protoreflect.Repeated {
				if arrayField != nil {
					return fmt.Errorf("multiple array fields")
				}

				arrayField = field
				continue
			}
			return fmt.Errorf("unknown field in response %s", field.Name())
		}

		if arrayField == nil {
			return fmt.Errorf("no array field")
		}

		if nextTokenField == nil {
			return fmt.Errorf("no next_token field")
		}

		arrayFieldOpt := arrayField.Options().(*descriptorpb.FieldOptions)
		validateOpt := proto.GetExtension(arrayFieldOpt, validate.E_Field).(*validate.FieldConstraints)
		if repeated := validateOpt.GetRepeated(); repeated != nil {
			if repeated.MaxItems != nil {
				pageSize = *repeated.MaxItems
			}
		}
	}

	var jsonRows = make([][]byte, 0, pageSize)

	selectQuery := sq.
		Select(fmt.Sprintf("%s.%s", StateAlias, gc.StateColumn)).
		From(fmt.Sprintf("%s AS %s", gc.StateTableName, StateAlias)).
		Limit(pageSize + 1)

	constrain(selectQuery)

	// TODO: Request Filters req := reqMsg.ProtoReflect()
	// TODO: Request Sorts
	// TODO: Pagination in Request

	var nextToken string
	if err := gc.DB.Transact(ctx, &sqrlx.TxOptions{
		ReadOnly:  true,
		Retryable: true,
		Isolation: sql.LevelReadCommitted,
	}, func(ctx context.Context, tx sqrlx.Transaction) error {
		rows, err := tx.Query(ctx, selectQuery)
		if err != nil {
			return fmt.Errorf("run select: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			var json []byte
			if err := rows.Scan(&json); err != nil {
				return err
			}
			jsonRows = append(jsonRows, json)
		}
		return rows.Err()
	}); err != nil {
		stmt, _, _ := selectQuery.ToSql()
		log.WithField(ctx, "query", stmt).Error("list query")
		return fmt.Errorf("list query: %w", err)
	}

	list := res.Mutable(arrayField).List()

	for idx, rowBytes := range jsonRows {
		rowMessage := list.NewElement().Message()
		if err := protojson.Unmarshal(rowBytes, rowMessage.Interface()); err != nil {
			return fmt.Errorf("unmarshal row: %w", err)
		}
		if idx >= int(pageSize) {
			// This is just pretend. The eventual solution will need to look at
			// the actual sorting and filtering of the query to determine the
			// next token.
			lastBytes, err := proto.Marshal(rowMessage.Interface())
			if err != nil {
				return fmt.Errorf("marshalling final row: %w", err)
			}
			nextToken = string(lastBytes)
			break
		}
		list.Append(protoreflect.ValueOf(rowMessage))
	}

	// TODO: Unclear why the Mutable doesn't work without this, but it doesn't
	// the result is an empty slice in the passed in object without this line
	res.Set(arrayField, protoreflect.ValueOf(list))

	if nextToken != "" {
		res.Set(nextTokenField, protoreflect.ValueOf(nextToken))
	}

	return nil

}

func (gc *GenericQuery) ListEvents(ctx context.Context, reqMsg proto.Message, resMsg proto.Message, constrain func(*sq.SelectBuilder)) error {

	var pageSize = uint64(20)
	var nextTokenField protoreflect.FieldDescriptor
	var arrayField protoreflect.FieldDescriptor
	res := resMsg.ProtoReflect()

	{ // Prepare
		// TODO: Build this on construction of GenericQuery

		resDesc := res.Descriptor()
		fields := resDesc.Fields()

		for i := 0; i < fields.Len(); i++ {
			field := fields.Get(i)
			if field.Name() == "next_token" {
				nextTokenField = field
				continue
			}
			if field.Cardinality() == protoreflect.Repeated {
				if arrayField != nil {
					return fmt.Errorf("multiple array fields")
				}

				arrayField = field
				continue
			}
			return fmt.Errorf("unknown field in response %s", field.Name())
		}

		if arrayField == nil {
			return fmt.Errorf("no array field")
		}

		if nextTokenField == nil {
			return fmt.Errorf("no next_token field")
		}

		arrayFieldOpt := arrayField.Options().(*descriptorpb.FieldOptions)
		validateOpt := proto.GetExtension(arrayFieldOpt, validate.E_Field).(*validate.FieldConstraints)
		if repeated := validateOpt.GetRepeated(); repeated != nil {
			if repeated.MaxItems != nil {
				pageSize = *repeated.MaxItems
			}
		}
	}

	var jsonRows = make([][]byte, 0, pageSize)

	selectQuery := sq.
		Select(fmt.Sprintf("%s AS event_data", gc.EventDataColumn)).
		From(fmt.Sprintf("%s AS state", gc.StateTableName)).
		LeftJoin(fmt.Sprintf(
			"%s AS event ON event.%s = state.%s",
			gc.EventTableName,
			gc.EventForeignKeyColumn,
			gc.PrimaryKeyColumn,
		)).
		Limit(pageSize + 1)

	constrain(selectQuery)

	// TODO: Request Filters req := reqMsg.ProtoReflect()
	// TODO: Request Sorts
	// TODO: Pagination in Request

	var nextToken string
	if err := gc.DB.Transact(ctx, &sqrlx.TxOptions{
		ReadOnly:  true,
		Retryable: true,
		Isolation: sql.LevelReadCommitted,
	}, func(ctx context.Context, tx sqrlx.Transaction) error {
		rows, err := tx.Query(ctx, selectQuery)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var json []byte
			if err := rows.Scan(&json); err != nil {
				return err
			}
			jsonRows = append(jsonRows, json)
		}
		return rows.Err()
	}); err != nil {
		return fmt.Errorf("tx: %w", err)
	}

	list := res.Mutable(arrayField).List()

	for idx, rowBytes := range jsonRows {
		rowMessage := list.NewElement().Message()
		if err := protojson.Unmarshal(rowBytes, rowMessage.Interface()); err != nil {
			return fmt.Errorf("unmarshal row: %w", err)
		}
		if idx >= int(pageSize) {
			// This is just pretend. The eventual solution will need to look at
			// the actual sorting and filtering of the query to determine the
			// next token.
			lastBytes, err := proto.Marshal(rowMessage.Interface())
			if err != nil {
				return err
			}
			nextToken = string(lastBytes)
			break
		}
		list.Append(protoreflect.ValueOf(rowMessage))
	}

	// TODO: Unclear why the Mutable doesn't work without this, but it doesn't
	// the result is an empty slice in the passed in object without this line
	res.Set(arrayField, protoreflect.ValueOf(list))

	if nextToken != "" {
		res.Set(nextTokenField, protoreflect.ValueOf(nextToken))
	}

	return nil

}
