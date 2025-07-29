package pgmigrate

import (
	"context"
	"fmt"
	"strings"

	sq "github.com/elgris/sqrl"
	"github.com/pentops/j5/lib/j5schema"
	"github.com/pentops/protostate/pquery"
	"github.com/pentops/sqrlx.go/sqrlx"
)

func IndexMigrations(spec pquery.TableSpec) ([]MigrationItem, error) {
	allMigrations := make([]MigrationItem, 0)
	indexes, err := buildIndexes(spec.TableName, spec.DataColumn, spec.RootObject)
	if err != nil {
		return nil, fmt.Errorf("building indexes: %w", err)
	}

	for _, index := range indexes {
		allMigrations = append(allMigrations, index)
	}

	return allMigrations, nil
}

func AddIndexes(ctx context.Context, conn sqrlx.Connection, specs ...pquery.TableSpec) error {
	allIndexes := make([]searchSpec, 0)
	for _, spec := range specs {
		indexes, err := buildIndexes(spec.TableName, spec.DataColumn, spec.RootObject)
		if err != nil {
			return err
		}
		allIndexes = append(allIndexes, indexes...)
	}

	return writeIndexes(ctx, conn, allIndexes)
}

type searchSpec struct {
	tsvColumn  string
	tableName  string
	columnName string
	path       pquery.Path
}

func buildIndexes(tableName string, columnName string, rootType *j5schema.ObjectSchema) ([]searchSpec, error) {

	cols, err := pquery.TSVColumns(rootType)
	if err != nil {
		return nil, err
	}

	specs := []searchSpec{}

	for _, col := range cols {
		specs = append(specs, searchSpec{
			tsvColumn:  col.ColumnName,
			tableName:  tableName,
			columnName: columnName,
			path:       col.Path,
		})

	}

	return specs, nil

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
