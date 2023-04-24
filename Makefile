DB_FILE=./whoishiring.db
MIGRATIONS_DIR=./migrations

.PHONY: run

run:
	go run . -serve

sync:
	go run . -sync

migrate-status:
	goose -dir $(MIGRATIONS_DIR) sqlite3 $(DB_FILE) status

migrate-up:
	goose -dir $(MIGRATIONS_DIR) sqlite3 $(DB_FILE) up

migrate-reset:
	goose -dir $(MIGRATIONS_DIR) sqlite3 $(DB_FILE) reset
