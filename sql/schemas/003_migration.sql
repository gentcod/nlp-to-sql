-- +goose Up
CREATE TYPE role_type AS ENUM ('user', 'admin');
ALTER TABLE auth ADD COLUMN role role_type DEFAULT 'user';

-- +goose Down
ALTER TABLE auth DROP COLUMN role;