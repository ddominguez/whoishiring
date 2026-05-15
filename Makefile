DB_FILE=./whoishiring.db
MIGRATIONS_DIR=./migrations

.PHONY: run

build:
	go build -o whoishiring

run:
	./whoishiring -serve

sync:
	./whoishiring -sync

verify:
	./whoishiring -verify

test:
	go test -v

migrate-status:
	goose -dir $(MIGRATIONS_DIR) sqlite3 $(DB_FILE) status

migrate-up:
	goose -dir $(MIGRATIONS_DIR) sqlite3 $(DB_FILE) up

migrate-create:
	goose -dir $(MIGRATIONS_DIR) sqlite3 $(DB_FILE) create $(NAME) sql
