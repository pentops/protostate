package psmpg

import (
	"context"
	"fmt"

	sq "github.com/elgris/sqrl"
	"github.com/pentops/j5/lib/j5query/pgmigrate"
	"github.com/pentops/protostate/psm"
	"github.com/pentops/sqrlx.go/sqrlx"
)

func BuildStateMachineMigrations(specs ...psm.QueryTableSpec) ([]byte, error) {

	allMigrations := make([]pgmigrate.MigrationItem, 0, len(specs)*4)

	for _, spec := range specs {
		if err := spec.Validate(); err != nil {
			return nil, fmt.Errorf("validate spec: %w", err)
		}
		stateTable, eventTable, err := BuildPSMTables(spec)
		if err != nil {
			return nil, err
		}
		allMigrations = append(allMigrations, stateTable, eventTable)

		stateList := spec.StateTable()
		indexes, err := pgmigrate.IndexMigrations(stateList)
		if err != nil {
			return nil, err
		}

		for _, index := range indexes {
			allMigrations = append(allMigrations, index)
		}

		eventList := spec.EventTable()
		indexes, err = pgmigrate.IndexMigrations(eventList)
		if err != nil {
			return nil, err
		}

		for _, index := range indexes {
			allMigrations = append(allMigrations, index)
		}
	}

	fileData, err := pgmigrate.PrintMigrations(allMigrations...)
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

	tables := make([]*pgmigrate.Table, 0, len(specs))
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

func AddIndexes(ctx context.Context, conn sqrlx.Transactor, specs ...psm.QueryTableSpec) error {
	indexes, err := BuildIndexes(specs...)
	if err != nil {
		return fmt.Errorf("build indexes: %w", err)
	}
	return pgmigrate.RunMigrations(ctx, conn, indexes)
}

func BuildIndexes(specs ...psm.QueryTableSpec) ([]pgmigrate.MigrationItem, error) {
	allIndexes := make([]pgmigrate.MigrationItem, 0)
	for _, spec := range specs {
		if err := spec.Validate(); err != nil {
			return nil, fmt.Errorf("validate spec: %w", err)
		}
		stateList := spec.StateTable()
		indexes, err := pgmigrate.IndexMigrations(stateList)
		if err != nil {
			return nil, err
		}
		allIndexes = append(allIndexes, indexes...)

		eventList := spec.EventTable()
		indexes, err = pgmigrate.IndexMigrations(eventList)
		if err != nil {
			return nil, err
		}
		allIndexes = append(allIndexes, indexes...)
	}

	return allIndexes, nil
}

func BuildPSMTables(spec psm.QueryTableSpec) (*pgmigrate.Table, *pgmigrate.Table, error) {

	stateTable := pgmigrate.CreateTable(spec.State.TableName)

	eventTable := pgmigrate.CreateTable(spec.Event.TableName).
		Column(spec.Event.ID.ColumnName, pgmigrate.UUID, pgmigrate.PrimaryKey)

	eventForeignKey := eventTable.ForeignKey("state", spec.State.TableName)
	for _, key := range spec.KeyColumns {
		format, err := pgmigrate.FieldFormat(key.Schema)
		if err != nil {
			return nil, nil, fmt.Errorf("key %s: %w", key.ColumnName, err)
		}

		if key.Primary {
			stateTable.Column(key.ColumnName, format, pgmigrate.PrimaryKey)
			eventTable.Column(key.ColumnName, format, pgmigrate.NotNull)
			eventForeignKey.Column(key.ColumnName, key.ColumnName)
			continue
		}
		if key.Required {
			stateTable.Column(key.ColumnName, format, pgmigrate.NotNull)
			eventTable.Column(key.ColumnName, format, pgmigrate.NotNull)
			continue
		}
		stateTable.Column(key.ColumnName, format)
		eventTable.Column(key.ColumnName, format)
	}

	stateTable.Column(spec.State.Root.ColumnName, pgmigrate.JSONB, pgmigrate.NotNull)

	eventTable.Column(spec.Event.Timestamp.ColumnName, pgmigrate.Timestamptz, pgmigrate.NotNull).
		Column(spec.Event.Sequence.ColumnName, pgmigrate.Int, pgmigrate.NotNull).
		Column(spec.Event.Root.ColumnName, pgmigrate.JSONB, pgmigrate.NotNull).
		Column(spec.Event.StateSnapshot.ColumnName, pgmigrate.JSONB, pgmigrate.NotNull)

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
