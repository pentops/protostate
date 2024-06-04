-- +goose Up

CREATE TABLE foo (
	foo_id uuid primary key,
	state jsonb NOT NULL,
	tenant_id uuid
);

CREATE TABLE foo_event (
	id uuid primary key,
	timestamp timestamptz NOT NULL,
  sequence int NOT NULL,
	foo_id uuid references foo(foo_id) NOT NULL,
	tenant_id uuid,
	data jsonb NOT NULL,
  state jsonb NOT NULL
);

CREATE TABLE bar (
	bar_id uuid,
  bar_other_id uuid,
	state jsonb NOT NULL,
  PRIMARY KEY (bar_id, bar_other_id)
);

CREATE TABLE bar_event (
	id uuid primary key,
	timestamp timestamptz NOT NULL,
  sequence int NOT NULL,
	bar_id uuid NOT NULL,
  bar_other_id uuid NOT NULL,
	data jsonb NOT NULL,
  state jsonb NOT NULL,
  CONSTRAINT fk_bar_event FOREIGN KEY (bar_id, bar_other_id) references bar(bar_id, bar_other_id )
);

-- +goose Down

DROP TABLE foo_event;
DROP TABLE foo;

DROP TABLE bar_event;
DROP TABLE bar;
