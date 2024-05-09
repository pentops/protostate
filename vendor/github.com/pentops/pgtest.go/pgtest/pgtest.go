package pgtest

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/lib/pq"
	"github.com/pressly/goose"
)

var shouldLog = os.Getenv("PGTEST_LOG") != ""

type CallbackConnector struct {
	*pq.Connector
	Callback func(context.Context, driver.Conn) error
}

func (tc *CallbackConnector) Connect(ctx context.Context) (driver.Conn, error) {
	conn, err := tc.Connector.Connect(ctx)
	if err != nil {
		return nil, err
	}
	if err := tc.Callback(ctx, conn); err != nil {
		return nil, err
	}
	return conn, nil
}

type options struct {
	url        string
	schemaName string
	dir        string
}

type option func(o *options)

func WithURL(url string) option {
	return func(o *options) {
		o.url = url
	}
}

func WithSchemaName(schemaName string) option {
	return func(o *options) {
		o.schemaName = schemaName
	}
}

func WithDir(dir string) option {
	return func(o *options) {
		o.dir = dir
	}
}

type TB interface {
	Fatal(...interface{})
}

func GetTestDB(t TB, optionMods ...option) *sql.DB {
	options := &options{
		schemaName: "testing",
		dir:        "./ext/db",
	}
	for _, mod := range optionMods {
		mod(options)
	}
	if options.url == "" {
		dbURL := os.Getenv("TEST_DB")
		if !strings.Contains(dbURL, "test") {
			t.Fatal("TEST_DB not a test database, it must contain the word 'test' somewhere in the connection string")
		}
		options.url = dbURL
	}

	conn, err := DialTestSchema(options.url, options.schemaName)
	if err != nil {
		t.Fatal(err.Error())
	}

	if err := migrate(conn, options.dir); err != nil {
		t.Fatal(err.Error())
	}
	return conn
}

type noLogger struct{}

func (*noLogger) Fatal(v ...interface{})                 {}
func (*noLogger) Fatalf(format string, v ...interface{}) {}
func (*noLogger) Print(v ...interface{})                 {}
func (*noLogger) Println(v ...interface{})               {}
func (*noLogger) Printf(format string, v ...interface{}) {}

func migrate(conn *sql.DB, dir string) error {
	if !shouldLog {
		goose.SetLogger(&noLogger{})
	}
	if err := goose.Up(conn, dir); err != nil {
		return err
	}
	return nil
}

func DialTestSchema(testURL string, name string) (*sql.DB, error) {

	connector, err := pq.NewConnector(testURL)
	if err != nil {
		return nil, err
	}

	conn := sql.OpenDB(connector)
	if err != nil {
		return nil, err
	}

	for tries := 0; tries < 30; tries++ {
		err = conn.Ping()
		if err == nil {
			break
		}
		time.Sleep(time.Second)
	}
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	if _, err := conn.ExecContext(ctx, fmt.Sprintf(`
		DROP SCHEMA IF EXISTS %s CASCADE;
		CREATE SCHEMA %s;
	`, name, name)); err != nil {
		return nil, err
	}
	conn.Close()

	testConnector := &CallbackConnector{
		Connector: connector,
		Callback: func(ctx context.Context, conn driver.Conn) error {
			execerCtx := conn.(driver.ExecerContext)
			_, err := execerCtx.ExecContext(ctx, fmt.Sprintf("SET search_path TO %s", name), []driver.NamedValue{})
			if err != nil {
				return fmt.Errorf("preparing connection to search_path: %w", err)
			}
			return nil
		},
	}

	return sql.OpenDB(testConnector), nil
}
