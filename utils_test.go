package main

import (
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/pressly/goose/v3"
)

// setupTestDB creates an in-memory sqlite3 database and applies available migrations.
func setupTestDB(t *testing.T) *sqlx.DB {
	migrationsPath := "./migrations"

	db, err := sqlx.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}

	goose.SetLogger(goose.NopLogger())

	err = goose.SetDialect(db.DriverName())
	if err != nil {
		t.Fatalf("failed to set dialect: %v", err)
	}

	err = goose.Up(db.DB, migrationsPath)
	if err != nil {
		t.Fatalf("failed to run migrations: %v", err)
	}

	return db
}
