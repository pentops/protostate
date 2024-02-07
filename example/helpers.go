package example

import (
	"context"

	"github.com/elgris/sqrl"
	"github.com/pentops/flowtest"
	"github.com/pentops/log.go/log"
	"github.com/pentops/sqrlx.go/sqrlx"
)

func silenceLogger() func() {
	defaultLogger := log.DefaultLogger
	log.DefaultLogger = log.NewCallbackLogger(func(level string, msg string, fields map[string]interface{}) {
	})
	return func() {
		log.DefaultLogger = defaultLogger
	}
}

func printQuery(t flowtest.TB, query *sqrl.SelectBuilder) {
	stmt, args, err := query.ToSql()
	if err != nil {
		t.Fatal(err.Error())
	}
	t.Log(stmt, args)
}

func getRawState(db *sqrlx.Wrapper, id string) (string, error) {
	var state []byte
	err := db.Transact(context.Background(), nil, func(ctx context.Context, tx sqrlx.Transaction) error {
		q := sqrl.Select("state").From("foo").Where("id = ?", id)
		err := tx.QueryRow(ctx, q).Scan(&state)
		if err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return "", err
	}

	return string(state), nil
}

func getRawEvent(db *sqrlx.Wrapper, id string) (string, error) {
	var data []byte
	err := db.Transact(context.Background(), nil, func(ctx context.Context, tx sqrlx.Transaction) error {
		q := sqrl.Select("data").From("foo_event").Where("id = ?", id)
		err := tx.QueryRow(ctx, q).Scan(&data)
		if err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return "", err
	}

	return string(data), nil
}
