package pgstore

import (
	"fmt"
)

type CreateTableBuilder struct {
	name    string
	columns []*Column
}

func CreateTable(name string) *CreateTableBuilder {
	return &CreateTableBuilder{
		name: name,
	}
}

func (t *CreateTableBuilder) Column(name string, typ ColumnType, flags ...ColumnFlag) *CreateTableBuilder {
	t.columns = append(t.columns, &Column{
		name:     name,
		typeName: typ,
		flags:    flags,
	})
	return t
}

type Column struct {
	name     string
	typeName ColumnType
	flags    []ColumnFlag
}

type ColumnFlag string

const (
	PrimaryKey ColumnFlag = "PRIMARY KEY"
	NotNull    ColumnFlag = "NOT NULL"
)

type ColumnType string

const (
	UUID        ColumnType = "uuid"
	Timestamptz ColumnType = "timestamptz"
	JSONB       ColumnType = "jsonb"
	Int         ColumnType = "int"
)

func PrintCreateMigration(tables ...*CreateTableBuilder) ([]byte, error) {
	p := newPrinter()
	p.p("-- +goose Up")
	p.setGap()
	for _, table := range tables {
		p.CreateTable(table)
	}
	p.p("-- +goose Down")
	p.setGap()
	for idx := len(tables) - 1; idx >= 0; idx-- {
		table := tables[idx]
		p.DropTable(table.name)
	}

	return p.bytes(), nil
}

type printer struct {
	buf []byte
	gap bool
}

func newPrinter() *printer {
	return &printer{}
}

func (p *printer) setGap() {
	p.gap = true
}

func (p *printer) p(elem ...interface{}) {
	if p.gap {
		p.buf = append(p.buf, '\n')
		p.gap = false
	}
	line := fmt.Sprintln(elem...)
	p.buf = append(p.buf, line...)
}

func (p *printer) bytes() []byte {
	return p.buf
}

func (p *printer) CreateTable(table *CreateTableBuilder) {
	p.p("CREATE TABLE ", table.name, " (")
	for i, col := range table.columns {
		if i > 0 {
			p.p(", ")
		}
		p.p(col.name, " ", col.typeName)
		for _, flag := range col.flags {
			p.p(" ", flag)
		}
	}
	p.p(");")
	p.setGap()
}

func (p *printer) DropTable(tableName string) {
	p.p("DROP TABLE ", tableName, ";")
	p.setGap()
}
