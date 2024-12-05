-- +goose Up
CREATE TABLE auth (
   id uuid PRIMARY KEY,
   email VARCHAR UNIQUE NOT NULL,
   harshed_password VARCHAR NOT NULL,
   password_changed_at TIMESTAMPTZ NOT NULL DEFAULT '0001-01-01 00:00:00Z',
   created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
   updated_at TIMESTAMPTZ NOT NULL DEFAULT '0001-01-01 00:00:00Z'
);

CREATE TABLE users(
   id uuid PRIMARY KEY,
   auth_id uuid NOT NULL REFERENCES auth(id),
   username VARCHAR UNIQUE NOT NULL,
   full_name VARCHAR NOT NULL,
   created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
   updated_at TIMESTAMPTZ NOT NULL DEFAULT '0001-01-01 00:00:00Z'
);

-- +goose Down
DROP TABLE users;
DROP TABLE auth;