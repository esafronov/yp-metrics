// Package includes pgx connection singleton for connection managing
package pg

import (
	"database/sql"

	_ "github.com/jackc/pgx/v5/stdlib"
)

var DB *sql.DB

// Open connection and sets DB variable
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
