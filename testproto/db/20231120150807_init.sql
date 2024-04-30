-- +goose Up

CREATE TABLE foo (
	id uuid primary key,
	state jsonb NOT NULL,
	tenant_id uuid,
	tsv_name tsvector GENERATED ALWAYS AS (
		to_tsvector('english', state ->> 'name')
	) STORED,
	tsv_field tsvector GENERATED ALWAYS AS (
		to_tsvector('english', state ->> 'field')
	) STORED,
	tsv_description tsvector GENERATED ALWAYS AS (
		to_tsvector('english', state ->> 'description')
	) STORED,
	tsv_profile_name tsvector GENERATED ALWAYS AS (
		to_tsvector('english', jsonb_path_query_array(state, '$.profiles[*].name'))
	) STORED
);

CREATE INDEX idx_tsv_name ON foo
USING gin(to_tsvector('english', state ->> 'name'));

CREATE INDEX idx_tsv_field ON foo
USING gin(to_tsvector('english', state ->> 'field'));

CREATE INDEX idx_tsv_description ON foo
USING gin(to_tsvector('english', state ->> 'description'));

CREATE INDEX idx_tsv_profile_name ON foo
USING gin(to_tsvector('english', jsonb_path_query_array(state, '$.profiles[*].name')));

CREATE TABLE foo_event (
	id uuid primary key,
	timestamp timestamptz NOT NULL,
	foo_id uuid references foo(id) NOT NULL,
	tenant_id uuid,
	actor jsonb NOT NULL,
	data jsonb NOT NULL,
  state jsonb NOT NULL
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

CREATE TABLE foo_cache (
  id uuid PRIMARY KEY,
  weight int NOT NULL,
  height int NOT NULL,
  length int NOT NULL
);

-- +goose Down

DROP TABLE foo_event;
DROP TABLE foo;

