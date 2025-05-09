package database

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

type DB struct {
	*sql.DB
}

func Init(databaseURL string) (*DB, error) {
	db, err := sql.Open("sqlite", databaseURL+"?_foreign_keys=on")
	if err != nil {
		return nil, fmt.Errorf("database connection failed: %w", err)
	}

	if err = db.Ping(); err != nil {
		return nil, fmt.Errorf("database ping failed: %w", err)
	}

	return &DB{db}, nil
}

func (db *DB) Close() {
	db.DB.Close()
}
