package dbconvert

import (
	"database/sql/driver"
	"fmt"
	"reflect"

	sq "github.com/elgris/sqrl"
	"github.com/pentops/j5/j5types/date_j5t"
	"github.com/pentops/j5/lib/j5codec"
	"github.com/pentops/j5/lib/j5reflect"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func FieldsToDBValues(m map[string]any) (map[string]any, error) {
	out := map[string]any{}
	for k, v := range m {
		converted, err := interfaceToDBValue(v)
		if err != nil {
			return nil, fmt.Errorf("field values: %w", err)
		}

		out[k] = converted
	}
	return out, nil
}

func FieldsToEqMap(ofTable string, m map[string]any) (sq.Eq, error) {
	out := sq.Eq{}
	for k, v := range m {
		converted, err := interfaceToDBValue(v)
		if err != nil {
			return nil, fmt.Errorf("eq map: %w", err)
		}

		fullKey := fmt.Sprintf("%s.%s", ofTable, k)
		out[fullKey] = converted
	}
	return out, nil
}

type j5message interface {
	J5Reflect() j5reflect.Root
}

func interfaceToDBValue(i any) (any, error) {
	switch v := i.(type) {
	case *timestamppb.Timestamp:
		return v.AsTime(), nil

	case *date_j5t.Date:
		return v.DateString(), nil

	case driver.Valuer:
		return v, nil

	case j5message:
		i, err := marshalJ5(v)
		if err != nil {
			return nil, fmt.Errorf("interface values: %w", err)
		}

		return i, nil
	case *string:
		if v == nil {
			return nil, nil
		}
		return v, nil
	}

	switch reflect.TypeOf(i).Kind() {
	case reflect.Ptr, reflect.Map, reflect.Chan, reflect.Slice, reflect.Array:
		if reflect.ValueOf(i).IsNil() {
			return nil, nil
		}
	}

	return i, nil
}

var codec *j5codec.Codec

func init() {
	codec = j5codec.NewCodec(j5codec.WithIncludeEmpty())
}

func MarshalJ5(msg j5reflect.Root) ([]byte, error) {
	return codec.ReflectToJSON(msg)
}

func UnmarshalJ5(b []byte, msg j5reflect.Root) error {
	return codec.JSONToReflect(b, msg)
}

func marshalJ5(msg j5message) (any, error) {
	if msg == nil {
		return nil, nil
	}

	j5r := msg.J5Reflect()
	if j5r == nil {
		return nil, fmt.Errorf("j5reflect is nil for %T", msg)
	}

	return MarshalJ5(j5r)
}

/*
func MarshalProto(state protoreflect.ProtoMessage) ([]byte, error) {
	// EmitDefaultValues behaves similarly to EmitUnpopulated, but does not emit "null"-value fields,
	// i.e. presence-sensing fields that are omitted will remain omitted to preserve presence-sensing.
	b, err := protojson.MarshalOptions{EmitDefaultValues: true}.Marshal(state)
	if err != nil {
		return b, fmt.Errorf("custom protomarshal: %w", err)
	}
	return b, nil
}*/
