// Package pg singleton includes Connect and Close functions for pgx driver of postgress
package pg

import (
	"database/sql"

	_ "github.com/jackc/pgx/v5/stdlib"
)

var DB *sql.DB

// Connect open connection and sets DB variable for singleton
func Connect(databaseDsn *string) error {
	db, err := sql.Open("pgx", *databaseDsn)
	if err != nil {
		return err
	}
	DB = db
	return nil
}

// Close connection DB
func Close() {
	if DB != nil {
		DB.Close()
	}
}
