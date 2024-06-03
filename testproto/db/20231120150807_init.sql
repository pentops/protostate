-- +goose Up

CREATE TABLE foo (
	foo_id uuid primary key,
	state jsonb NOT NULL,
	tenant_id uuid,
	tsv_name tsvector GENERATED ALWAYS AS (
		to_tsvector('english', state -> 'data' ->> 'name')
	) STORED,
	tsv_field tsvector GENERATED ALWAYS AS (
		to_tsvector('english', state -> 'data' ->> 'field')
	) STORED,
	tsv_description tsvector GENERATED ALWAYS AS (
		to_tsvector('english', state -> 'data' ->> 'description')
	) STORED,
	tsv_profile_name tsvector GENERATED ALWAYS AS (
		to_tsvector('english', jsonb_path_query_array(state -> 'data' , '$.profiles[*].name'))
	) STORED
);

CREATE INDEX idx_tsv_name ON foo
USING gin(to_tsvector('english', state -> 'data' ->> 'name'));

CREATE INDEX idx_tsv_field ON foo
USING gin(to_tsvector('english', state -> 'data' ->> 'field'));

CREATE INDEX idx_tsv_description ON foo
USING gin(to_tsvector('english', state -> 'data' ->> 'description'));

CREATE INDEX idx_tsv_profile_name ON foo
USING gin(to_tsvector('english', jsonb_path_query_array(state -> 'data', '$.profiles[*].name')));

CREATE TABLE foo_event (
	id uuid primary key,
	timestamp timestamptz NOT NULL,
  sequence int NOT NULL,
	foo_id uuid references foo(foo_id) NOT NULL,
	tenant_id uuid,
	cause jsonb NOT NULL,
	data jsonb NOT NULL,
  state jsonb NOT NULL
);

CREATE TABLE bar (
	bar_id uuid primary key,
	state jsonb NOT NULL
);

CREATE TABLE bar_event (
	id uuid primary key,
	timestamp timestamptz NOT NULL,
  sequence int NOT NULL,
	bar_id uuid references bar(bar_id) NOT NULL,
	cause jsonb NOT NULL,
	data jsonb NOT NULL,
  state jsonb NOT NULL
);

CREATE TABLE foo_cache (
  id uuid PRIMARY KEY,
  weight int NOT NULL,
  height int NOT NULL,
  length int NOT NULL
);

-- +goose Down

DROP TABLE foo_event;
DROP TABLE foo;

