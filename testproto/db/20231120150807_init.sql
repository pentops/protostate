-- +goose Up

CREATE TABLE foo (
	id uuid primary key,
	state jsonb NOT NULL,
	tenant_id uuid,
	foo_name_tsv_1 tsvector GENERATED ALWAYS AS (
		to_tsvector('english', state ->> 'name')
	) STORED,
	foo_field_tsv_1 tsvector GENERATED ALWAYS AS (
		to_tsvector('english', state ->> 'field')
	) STORED,
	foo_description_tsv_1 tsvector GENERATED ALWAYS AS (
		to_tsvector('english', state ->> 'description')
	) STORED,
	profile_name_tsv_1 tsvector GENERATED ALWAYS AS (
		to_tsvector('english', jsonb_path_query_array(state, '$.profiles[*].name'))
	) STORED
);

CREATE INDEX idx_foo_name_tsv_1 ON foo
USING gin(to_tsvector('english', state ->> 'name'));

CREATE INDEX idx_foo_field_tsv_1 ON foo
USING gin(to_tsvector('english', state ->> 'field'));

CREATE INDEX idx_foo_description_tsv_1 ON foo
USING gin(to_tsvector('english', state ->> 'description'));

CREATE INDEX idx_profile_name_tsv_1 ON foo
USING gin(to_tsvector('english', jsonb_path_query_array(state, '$.profiles[*].name')));

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

