// Package pg singleton includes Connect and Close functions for pgx driver of postgress
package pg

import (
	"database/sql"
	"errors"

	"github.com/esafronov/yp-metrics/internal/logger"
	_ "github.com/jackc/pgx/v5/stdlib"
)

var DB *sql.DB

// Connect open connection and sets DB variable for singleton
func Connect(databaseDsn *string) error {
	if databaseDsn == nil {
		return errors.New("databaseDsn is nil")
	}
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
		err := DB.Close()
		if err != nil {
			logger.Log.Info(err.Error())
		}
	}
}
