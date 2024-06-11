-- +goose Up

CREATE TABLE foo_cache (
  id uuid PRIMARY KEY,
  weight int NOT NULL,
  height int NOT NULL,
  length int NOT NULL
);

-- +goose Down

DROP TABLE foo_cache;

