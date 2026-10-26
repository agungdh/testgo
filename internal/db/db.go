package db

import (
	"database/sql"
	"os"
	"time"

	_ "github.com/pressly/goose/v3"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
)

// NewDB membuat koneksi *bun.DB dari env DATABASE_URL.
// Jika DATABASE_URL kosong, pakai default localhost.
func NewDB() *bun.DB {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://postgres:postgres@localhost:5432/testgo?sslmode=disable"
	}
	sqldb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(dsn)))
	// VPS 1 core / 1GB RAM: pool kecil agar tidak makan memori.
	sqldb.SetMaxOpenConns(10)
	sqldb.SetMaxIdleConns(2)
	sqldb.SetConnMaxLifetime(5 * time.Minute)
	sqldb.SetConnMaxIdleTime(1 * time.Minute)
	return bun.NewDB(sqldb, pgdialect.New())
}