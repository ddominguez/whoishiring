DB_FILE=./whoishiring.db
MIGRATIONS_DIR=./migrations

.PHONY: run migrate-status migrate-up

run:
	go run main.go

migrate-status:
	goose -dir $(MIGRATIONS_DIR) sqlite3 $(DB_FILE) status

migrate-up:
	goose -dir $(MIGRATIONS_DIR) sqlite3 $(DB_FILE) up

migrate-reset:
	goose -dir $(MIGRATIONS_DIR) sqlite3 $(DB_FILE) reset
