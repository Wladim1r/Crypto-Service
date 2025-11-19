// Package db
package db

import (
	"database/sql"
	"log/slog"
	"os"

	_ "modernc.org/sqlite"
)

func MustLoad() *sql.DB {
	db, err := sql.Open("sqlite", "./users.db")
	if err != nil {
		slog.Error("Could not open db sqlite", "error", err)
		os.Exit(1)
	}

	return db
}
