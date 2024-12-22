-- +goose Up
CREATE TABLE admins(
   id uuid PRIMARY KEY,
   auth_id uuid NOT NULL REFERENCES auth(id),
   username VARCHAR UNIQUE NOT NULL,
   full_name VARCHAR NOT NULL,
   created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
   updated_at TIMESTAMPTZ NOT NULL DEFAULT '0001-01-01 00:00:00Z'
);

-- +goose Down
DROP TABLE admins;