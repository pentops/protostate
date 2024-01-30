-- +goose Up

CREATE TABLE foo (
	id uuid primary key,
	state jsonb NOT NULL,
	tenant_id uuid
);

CREATE TABLE foo_event (
	id uuid primary key,
	timestamp timestamptz NOT NULL,
	foo_id uuid references foo(id) NOT NULL,
	tenant_id uuid,
	actor jsonb NOT NULL,
	data jsonb NOT NULL
);

CREATE TABLE bar (
	id uuid primary key,
	state jsonb NOT NULL
);

CREATE TABLE bar_event (
	id uuid primary key,
	timestamp timestamptz NOT NULL,
	bar_id uuid references bar(id) NOT NULL,
	data jsonb NOT NULL
);

-- +goose Down

DROP TABLE foo_event;
DROP TABLE foo;

