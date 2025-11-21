// Package db
package db

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/Wladim1r/auth/lib/getenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func MustLoad() *gorm.DB {
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=5432 sslmode=disable",
		getenv.GetString("POSTGRES_HOST", "postgres"),
		getenv.GetString("POSTGRES_USER", "postgres"),
		getenv.GetString("POSTGRES_PASSWORD", "postgres"),
		getenv.GetString("POSTGRES_DB", "users"),
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		slog.Error("Could not connect to db", "error", err.Error())
		os.Exit(1)
	}

	return db
}
