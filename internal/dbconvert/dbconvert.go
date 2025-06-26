package dbconvert

import (
	"database/sql/driver"
	"fmt"
	"reflect"

	sq "github.com/elgris/sqrl"
	"github.com/pentops/j5/j5types/date_j5t"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
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

func interfaceToDBValue(i any) (any, error) {
	fmt.Printf("interfaceToDBValue: %T\n", i)
	switch v := i.(type) {
	case *timestamppb.Timestamp:
		return v.AsTime(), nil

	case *date_j5t.Date:
		fmt.Printf("date: %v\n", v)
		return v.DateString(), nil

	case driver.Valuer:
		return v, nil

	case proto.Message:
		i, err := MarshalProto(v)
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

func MarshalProto(state protoreflect.ProtoMessage) ([]byte, error) {
	// EmitDefaultValues behaves similarly to EmitUnpopulated, but does not emit "null"-value fields,
	// i.e. presence-sensing fields that are omitted will remain omitted to preserve presence-sensing.
	b, err := protojson.MarshalOptions{EmitDefaultValues: true}.Marshal(state)
	if err != nil {
		return b, fmt.Errorf("custom protomarshal: %w", err)
	}
	return b, nil
}
