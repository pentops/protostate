-- +goose Up

ALTER TABLE foo ADD COLUMN tsv_name tsvector GENERATED ALWAYS AS (
		to_tsvector('english', state -> 'data' ->> 'name')
	) STORED;

CREATE INDEX idx_tsv_name ON foo
USING gin(to_tsvector('english', state -> 'data' ->> 'name'));

ALTER TABLE foo ADD COLUMN tsv_field tsvector GENERATED ALWAYS AS (
		to_tsvector('english', state -> 'data' ->> 'field')
	) STORED;

CREATE INDEX idx_tsv_field ON foo
USING gin(to_tsvector('english', state -> 'data' ->> 'field'));

ALTER TABLE foo ADD COLUMN tsv_description tsvector GENERATED ALWAYS AS (
		to_tsvector('english', state -> 'data' ->> 'description')
	) STORED;

CREATE INDEX idx_tsv_description ON foo
USING gin(to_tsvector('english', state -> 'data' ->> 'description'));

ALTER TABLE foo ADD COLUMN tsv_profile_name tsvector GENERATED ALWAYS AS (
		to_tsvector('english', jsonb_path_query_array(state -> 'data' , '$.profiles[*].name'))
	) STORED;

CREATE INDEX idx_tsv_profile_name ON foo
USING gin(to_tsvector('english', jsonb_path_query_array(state -> 'data', '$.profiles[*].name')));

-- +goose Down
