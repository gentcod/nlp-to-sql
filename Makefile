current_dir = $(shell pwd)
sqlc:
	docker run --rm -v $(current_dir):/src -w /src sqlc/sqlc generate

createdb:
	docker exec -it postgres12 createdb --username=root --owner=root nql_to_sql

dropdb:
	docker exec -it postgres12 dropdb nql_to_sql

gooseup:
	goose -dir sql/schemas postgres postgres://root:secret@localhost:5431/dummy_bank?sslmode=disable up

goosedown:
	goose -dir sql/schemas postgres postgres://root:secret@localhost:5431/dummy_bank?sslmode=disable down

test:
	go test -v -cover -short ./...

mock:
	mockgen -package mockdb -destination internal/database/mock/store.go github.com/gentcod/DummyBank/internal/database Store

buildimage:
	docker build -t nlqtosql:latest .

.PHONY: sqlc createdb dropdb gooseup goosedown test mock buildimage