package pgmigrate

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	"github.com/pentops/sqrlx.go/sqrlx"
)

type MigrationItem interface {
	ToSQL() (string, error)
	DownSQL() (string, error)
}

func RunMigrations(ctx context.Context, db sqrlx.Transactor, migrations []MigrationItem) error {

	return db.Transact(ctx, nil, func(ctx context.Context, tx sqrlx.Transaction) error {
		for _, migration := range migrations {
			statement, err := migration.ToSQL()
			if err != nil {
				return err
			}
			if _, err := tx.ExecRaw(ctx, statement); err != nil {
				return err
			}
		}
		return nil
	})
}

func PrintMigrations(items ...MigrationItem) ([]byte, error) {
	p := newPrinter()
	p.p("-- +goose Up")
	p.setGap()
	for _, table := range items {
		val, err := table.ToSQL()
		if err != nil {
			return nil, err
		}
		p.p(val)
		p.setGap()
	}
	p.p("-- +goose Down")
	p.setGap()
	for idx := len(items) - 1; idx >= 0; idx-- {
		item := items[idx]
		downVal, err := item.DownSQL()
		if err != nil {
			return nil, err
		}
		if len(strings.Split(downVal, "\n")) > 1 {
			p.setGap()
			p.p(downVal)
			p.setGap()
		} else {
			p.p(downVal)
		}
	}

	return p.bytes(), nil
}

type printer struct {
	buf bytes.Buffer
	gap bool
}

func newPrinter() *printer {
	return &printer{
		buf: bytes.Buffer{},
	}
}

func (p *printer) setGap() {
	p.gap = true
}

func (p *printer) p(elem ...any) {
	if p.gap {
		fmt.Fprintln(&p.buf)
		p.gap = false
	}
	for _, elem := range elem {
		fmt.Fprint(&p.buf, elem)
	}
	fmt.Fprintln(&p.buf)
}

func (p *printer) bytes() []byte {
	return p.buf.Bytes()
}
