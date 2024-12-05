current_dir = $(shell pwd)
run:
	go run .

build:
	go build -o bin/nlptosql .

sqlc:
	docker run --rm -v $(current_dir):/src -w /src sqlc/sqlc generate

createdb:
	docker exec -it postgres12 createdb --username=root --owner=root dbchat

dropdb:
	docker exec -it postgres12 dropdb dbchat

gooseup:
	goose -dir sql/schemas postgres postgres://root:secret@localhost:5431/dbchat?sslmode=disable up

goosedown:
	goose -dir sql/schemas postgres postgres://root:secret@localhost:5431/dbchat?sslmode=disable down

test:
	go test -v -cover -short ./...

mock:
	mockgen -package mockdb -destination internal/database/mock/store.go github.com/gentcod/nlp-to-sql/internal/database Store

buildimage:
	docker build -t nlqtosql:latest .

.PHONY: run build sqlc createdb dropdb gooseup goosedown test mock buildimage