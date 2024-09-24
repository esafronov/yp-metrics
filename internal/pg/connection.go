package pg

import (
	"database/sql"

	_ "github.com/jackc/pgx/v5/stdlib"
)

var DB *sql.DB

func Connect(databaseDsn *string) error {
	db, err := sql.Open("pgx", *databaseDsn)
	if err != nil {
		return err
	}
	DB = db
	return nil
}

func Close() {
	if DB != nil {
		DB.Close()
	}
}
