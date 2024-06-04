package pgmigrate

import (
	"bytes"
	"fmt"
	"strings"
)

type CreateTableBuilder struct {
	name        string
	columns     []*column
	foreignKeys []*ForeignKeyBuilder
}

func CreateTable(name string) *CreateTableBuilder {
	return &CreateTableBuilder{
		name: name,
	}
}

func (t *CreateTableBuilder) Column(name string, typ ColumnType, options ...ColumnOption) *CreateTableBuilder {
	column := &column{
		name:     name,
		typeName: typ,
		flags:    []string{},
	}
	for _, opt := range options {
		opt(column)
	}
	t.columns = append(t.columns, column)
	return t
}

func (t *CreateTableBuilder) ForeignKey(name, tableName string) *ForeignKeyBuilder {
	fk := &ForeignKeyBuilder{
		name:      name,
		tableName: tableName,
	}
	t.foreignKeys = append(t.foreignKeys, fk)
	return fk
}

type ForeignKeyBuilder struct {
	name      string
	tableName string
	columns   []ColumnPair
}

type ColumnPair struct {
	local, foreign string
}

func (fk *ForeignKeyBuilder) Column(localName, remoteName string) *ForeignKeyBuilder {
	fk.columns = append(fk.columns, ColumnPair{local: localName, foreign: remoteName})
	return fk
}

type column struct {
	name string

	primaryKey bool // Multi Primary Key is possible
	notNull    bool

	typeName ColumnType
	flags    []string
}

type ColumnOption func(*column)

// PrimaryKey adds this column as a primary key, if there are multiple
// primary keys, they will be added as a composite key
func PrimaryKey(c *column) {
	c.primaryKey = true
}

func NotNull(c *column) {
	c.notNull = true
}

type ColumnType string

const (
	UUID        ColumnType = "uuid"
	Timestamptz ColumnType = "timestamptz"
	JSONB       ColumnType = "jsonb"
	Int         ColumnType = "int"
)

type Table struct {
	Name        string
	Columns     []Column
	PrimaryKey  []string
	ForeignKeys []ForeignKey
}

type Column struct {
	Name  string
	Type  string
	Flags []string
}

type ForeignKey struct {
	Name      string
	TableName string
	Columns   []ColumnPair
}

func (t *CreateTableBuilder) Build() (*Table, error) {
	table := &Table{
		Name: t.name,
	}

	for _, col := range t.columns {
		column := Column{
			Name: col.name,
			Type: string(col.typeName),
		}
		if col.primaryKey {
			table.PrimaryKey = append(table.PrimaryKey, col.name)
		}
		if col.notNull {
			column.Flags = append(column.Flags, "NOT NULL")
		}
		table.Columns = append(table.Columns, column)
	}

	for _, fk := range t.foreignKeys {
		foreignKey := ForeignKey{
			Name:      fk.name,
			TableName: fk.tableName,
			Columns:   fk.columns,
		}

		table.ForeignKeys = append(table.ForeignKeys, foreignKey)
	}

	return table, nil
}

func (t *CreateTableBuilder) ToSQL() (string, error) {
	p := newPrinter()
	if err := p.CreateTable(t); err != nil {
		return "", err
	}

	return string(p.bytes()), nil
}

func PrintCreateMigration(tables ...*CreateTableBuilder) ([]byte, error) {
	p := newPrinter()
	p.p("-- +goose Up")
	p.setGap()
	for _, table := range tables {
		if err := p.CreateTable(table); err != nil {
			return nil, err
		}
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

func (p *printer) p(elem ...interface{}) {
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

func (p *printer) CreateTable(builder *CreateTableBuilder) error {

	table, err := builder.Build()
	if err != nil {
		return err
	}

	p.p("CREATE TABLE ", table.Name, " (")

	clauses := make([]string, 0)

	for _, col := range table.Columns {
		line := make([]string, 2+len(col.Flags))
		line[0] = col.Name
		line[1] = col.Type
		copy(line[2:], col.Flags)
		clauses = append(clauses, strings.Join(line, " "))
	}

	if len(table.PrimaryKey) > 0 {
		clauses = append(clauses, fmt.Sprintf("CONSTRAINT %s_pk PRIMARY KEY (%s)", table.Name, strings.Join(table.PrimaryKey, ", ")))
	}

	for _, fk := range table.ForeignKeys {
		localColumns := make([]string, 0, len(fk.Columns))
		remoteColumns := make([]string, 0, len(fk.Columns))
		for _, col := range fk.Columns {
			localColumns = append(localColumns, col.local)
			remoteColumns = append(remoteColumns, col.foreign)
		}

		clauses = append(clauses, fmt.Sprintf("CONSTRAINT %s_fk_%s FOREIGN KEY (%s) REFERENCES %s(%s)", table.Name, fk.Name, strings.Join(localColumns, ", "), fk.TableName, strings.Join(remoteColumns, ", ")))
	}

	for idx, clause := range clauses {
		suffix := ","
		if idx == len(clauses)-1 {
			suffix = ""
		}
		p.p("  ", clause, suffix)
	}
	p.p(");")
	p.setGap()
	return nil
}

func (p *printer) DropTable(tableName string) {
	p.p("DROP TABLE ", tableName, ";")
	p.setGap()
}
