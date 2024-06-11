package pgmigrate

import (
	"context"
	"fmt"
	"strings"

	sq "github.com/elgris/sqrl"
	"github.com/pentops/protostate/gen/list/v1/psml_pb"
	"github.com/pentops/protostate/internal/pgstore"
	"github.com/pentops/protostate/psm"
	"github.com/pentops/sqrlx.go/sqrlx"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
)

func BuildStateMachineMigrations(specs ...psm.QueryTableSpec) ([]byte, error) {

	allMigrations := make([]MigrationItem, 0, len(specs)*4)

	for _, spec := range specs {
		stateTable, eventTable, err := BuildPSMTables(spec)
		if err != nil {
			return nil, err
		}
		allMigrations = append(allMigrations, stateTable)

		indexes, err := buildIndexes(spec.State.TableName, spec.State.Root.ColumnName, spec.StateType)
		if err != nil {
			return nil, err
		}
		for _, index := range indexes {
			allMigrations = append(allMigrations, index)
		}

		allMigrations = append(allMigrations, eventTable)

		indexes, err = buildIndexes(spec.Event.TableName, spec.Event.Root.ColumnName, spec.EventType)
		if err != nil {
			return nil, err
		}
		for _, index := range indexes {
			allMigrations = append(allMigrations, index)
		}
	}

	fileData, err := PrintMigrations(allMigrations...)
	if err != nil {
		return nil, err
	}
	return fileData, nil
}

func CreateStateMachines(ctx context.Context, conn sqrlx.Connection, specs ...psm.QueryTableSpec) error {
	db, err := sqrlx.New(conn, sq.Dollar)
	if err != nil {
		return err
	}

	tables := make([]*Table, 0, len(specs))
	for _, spec := range specs {
		stateTable, eventTable, err := BuildPSMTables(spec)
		if err != nil {
			return err
		}
		tables = append(tables, stateTable, eventTable)
	}

	return db.Transact(ctx, nil, func(ctx context.Context, tx sqrlx.Transaction) error {
		if _, err := conn.BeginTx(ctx, nil); err != nil {
			return err
		}

		for _, table := range tables {

			statement, err := table.ToSQL()
			if err != nil {
				return err
			}
			_, err = tx.ExecRaw(ctx, statement)
			if err != nil {
				return err
			}
		}
		return nil
	})
}

type searchSpec struct {
	tsvColumn  string
	tableName  string
	columnName string
	path       pgstore.Path
}

func (ss searchSpec) ToSQL() (string, error) {
	statement := fmt.Sprintf("to_tsvector('english', jsonb_path_query_array(%s, '%s'))", ss.columnName, ss.path.JSONPathQuery())

	lines := []string{}

	lines = append(lines, fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s tsvector GENERATED ALWAYS", ss.tableName, ss.tsvColumn))
	lines = append(lines, fmt.Sprintf("  AS (%s) STORED;", statement))
	lines = append(lines, "")
	lines = append(lines, fmt.Sprintf("CREATE INDEX %s_%s_idx ON %s USING GIN (%s);", ss.tableName, ss.tsvColumn, ss.tableName, ss.tsvColumn))
	return strings.Join(lines, "\n"), nil

}

func (ss searchSpec) DownSQL() (string, error) {
	return fmt.Sprintf("DROP INDEX %s_%s_idx;\nALTER TABLE %s DROP COLUMN %s;", ss.tableName, ss.tsvColumn, ss.tableName, ss.tsvColumn), nil
}

func AddIndexes(ctx context.Context, conn sqrlx.Connection, specs ...psm.QueryTableSpec) error {
	allIndexes := make([]searchSpec, 0)
	for _, spec := range specs {
		indexes, err := buildIndexes(spec.State.TableName, spec.State.Root.ColumnName, spec.StateType)
		if err != nil {
			return err
		}
		allIndexes = append(allIndexes, indexes...)

		indexes, err = buildIndexes(spec.Event.TableName, spec.Event.Root.ColumnName, spec.EventType)
		if err != nil {
			return err
		}
		allIndexes = append(allIndexes, indexes...)
	}

	return writeIndexes(ctx, conn, allIndexes)
}

func buildIndexes(tableName string, columnName string, rootType protoreflect.MessageDescriptor) ([]searchSpec, error) {

	specs := []searchSpec{}

	if err := pgstore.WalkPathNodes(rootType, func(node pgstore.Path) error {
		field := node.LeafField()
		if field == nil || field.Kind() != protoreflect.StringKind {
			return nil
		}

		fieldOpts, ok := proto.GetExtension(field.Options().(*descriptorpb.FieldOptions), psml_pb.E_Field).(*psml_pb.FieldConstraint)
		if !ok {
			return nil
		}

		switch fieldOpts.GetString_().GetWellKnown().(type) {
		case *psml_pb.StringRules_OpenText:
			searchOpts := fieldOpts.GetString_().GetOpenText().GetSearching()
			if searchOpts == nil || !searchOpts.Searchable {
				return nil
			}

			specs = append(specs, searchSpec{
				tsvColumn:  searchOpts.GetFieldIdentifier(),
				tableName:  tableName,
				columnName: columnName,
				path:       node,
			})
		}

		return nil
	}); err != nil {
		return nil, err
	}

	return specs, nil

}

func writeIndexes(ctx context.Context, conn sqrlx.Connection, specs []searchSpec) error {

	db, err := sqrlx.New(conn, sq.Dollar)
	if err != nil {
		return err
	}

	return db.Transact(ctx, nil, func(ctx context.Context, tx sqrlx.Transaction) error {
		for _, spec := range specs {

			var count int

			err := tx.QueryRow(ctx, sq.Select("COUNT(column_name)").
				From("information_schema.columns").
				Where("table_schema = CURRENT_SCHEMA").
				Where(sq.Eq{"table_name": spec.tableName, "column_name": spec.tsvColumn})).Scan(&count)
			if err != nil {
				return err
			}
			if count > 0 {
				return nil
			}

			statement := fmt.Sprintf("to_tsvector('english', jsonb_path_query_array(%s, '%s'))", spec.columnName, spec.path.JSONPathQuery())

			_, err = tx.ExecRaw(ctx, fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s tsvector GENERATED ALWAYS AS (%s) STORED;", spec.tableName, spec.tsvColumn, statement))
			if err != nil {
				return err
			}

			_, err = tx.ExecRaw(ctx, fmt.Sprintf("CREATE INDEX %s_%s_idx ON %s USING GIN (%s);", spec.tableName, spec.tsvColumn, spec.tableName, spec.tsvColumn))
			if err != nil {
				return err
			}

		}
		return nil
	})

}

func BuildPSMTables(spec psm.QueryTableSpec) (*Table, *Table, error) {

	stateTable := CreateTable(spec.State.TableName)

	eventTable := CreateTable(spec.Event.TableName).
		Column(spec.Event.ID.ColumnName, "uuid", PrimaryKey)

	eventForeignKey := eventTable.ForeignKey("state", spec.State.TableName)
	for _, key := range spec.KeyColumns {
		if key.Primary {
			stateTable.Column(key.ColumnName, "uuid", PrimaryKey)
			eventTable.Column(key.ColumnName, "uuid", NotNull)
			eventForeignKey.Column(key.ColumnName, key.ColumnName)
			continue
		}
		if key.Required {
			stateTable.Column(key.ColumnName, "uuid", NotNull)
			eventTable.Column(key.ColumnName, "uuid", NotNull)
			continue
		}
		stateTable.Column(key.ColumnName, "uuid")
		eventTable.Column(key.ColumnName, "uuid")
	}

	stateTable.Column(spec.State.Root.ColumnName, "jsonb", NotNull)

	eventTable.Column(spec.Event.Timestamp.ColumnName, "timestamptz", NotNull).
		Column(spec.Event.Sequence.ColumnName, "int", NotNull).
		Column(spec.Event.Root.ColumnName, "jsonb", NotNull).
		Column(spec.Event.StateSnapshot.ColumnName, "jsonb", NotNull)

	state, err := stateTable.Build()
	if err != nil {
		return nil, nil, err
	}

	event, err := eventTable.Build()
	if err != nil {
		return nil, nil, err
	}

	return state, event, nil
}
