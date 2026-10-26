package db

import (
	"database/sql"

	"testgo/internal/config"

	_ "github.com/pressly/goose/v3"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
)

// NewDB membuat koneksi *bun.DB dari konfigurasi aplikasi.
// Konfigurasi dibaca dari .env / environment oleh config.Load().
func NewDB(cfg config.Config) *bun.DB {
	sqldb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(cfg.DatabaseURL)))
	// VPS 1 core / 1GB RAM: pool kecil agar tidak makan memori.
	sqldb.SetMaxOpenConns(cfg.DB.MaxOpenConns)
	sqldb.SetMaxIdleConns(cfg.DB.MaxIdleConns)
	sqldb.SetConnMaxLifetime(cfg.DB.ConnMaxLifetime)
	sqldb.SetConnMaxIdleTime(cfg.DB.ConnMaxIdleTime)
	return bun.NewDB(sqldb, pgdialect.New())
}
