package pg

import (
	"database/sql"
	"fmt"

	_ "github.com/jackc/pgx/v5/stdlib"
)

var DB *sql.DB

func Connect(databaseDsn *string) error {
	if *databaseDsn == "" {
		return fmt.Errorf("databaseDsn shoud not be empty")
	}
	db, err := sql.Open("pgx", *databaseDsn)
	if err != nil {
		return err
	}
	DB = db
	return nil
}

func Close() {
	DB.Close()
}
