package pquery

import (
	"fmt"

	"github.com/pentops/j5/lib/j5reflect"
	"github.com/pentops/j5/lib/j5schema"
)

func assertSchemaMatch(wantSchema *j5schema.ObjectSchema, gotObject j5reflect.Object) error {
	wantSchemaName := wantSchema.FullName()
	gotSchemaName := gotObject.SchemaName()
	if wantSchemaName != gotSchemaName {
		return fmt.Errorf("provided object is type %s, expects %s",
			gotSchemaName,
			wantSchemaName,
		)
	}
	return nil
}
func assertObjectsMatch(ms *j5schema.MethodSchema, req, res j5reflect.Object) error {
	if err := assertSchemaMatch(ms.Request, req); err != nil {
		return fmt.Errorf("request object: %w", err)
	}
	if err := assertSchemaMatch(ms.Response, res); err != nil {
		return fmt.Errorf("response object: %w", err)
	}

	return nil
}
