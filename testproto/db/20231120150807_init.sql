-- +goose Up

CREATE TABLE foo (
	id uuid primary key,
	state jsonb,
	tenant_id uuid
);

CREATE TABLE foo_event (
	id uuid primary key,
	foo_id uuid references foo(id),
	data jsonb
);

-- +goose Down

DROP TABLE foo_event;
DROP TABLE foo;

