package dbconvert

import (
	"fmt"
	"reflect"

	sq "github.com/elgris/sqrl"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func FieldsToDBValues(m map[string]interface{}) (map[string]interface{}, error) {
	out := map[string]interface{}{}
	for k, v := range m {
		converted, err := interfaceToDBValue(v)
		if err != nil {
			return nil, err
		}

		out[k] = converted
	}
	return out, nil
}

func FieldsToEqMap(ofTable string, m map[string]interface{}) (sq.Eq, error) {
	out := sq.Eq{}
	for k, v := range m {
		converted, err := interfaceToDBValue(v)
		if err != nil {
			return nil, err
		}

		fullKey := fmt.Sprintf("%s.%s", ofTable, k)
		out[fullKey] = converted
	}
	return out, nil
}

func interfaceToDBValue(i interface{}) (interface{}, error) {
	switch v := i.(type) {
	case *timestamppb.Timestamp:
		return v.AsTime(), nil
	case proto.Message:
		return protojson.Marshal(v)
	case *string:
		if v == nil {
			return nil, nil
		}
		return v, nil
	}

	switch reflect.TypeOf(i).Kind() {
	case reflect.Ptr, reflect.Map, reflect.Array, reflect.Chan, reflect.Slice:
		if reflect.ValueOf(i).IsNil() {
			return nil, nil
		}
	}

	return i, nil
}
