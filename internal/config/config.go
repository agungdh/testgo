// Package config memuat konfigurasi aplikasi dari file .env (opsional)
// yang digabung dengan environment proses.
package config

import (
	"errors"
	"log/slog"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

// Config berisi seluruh konfigurasi aplikasi.
type Config struct {
	AppEnv      string
	Port        string
	DatabaseURL string
	DB          DBConfig
}

// DBConfig berisi pengaturan connection pool database.
type DBConfig struct {
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
}

// Load membaca .env (jika ada) lalu menggabungkannya dengan environment proses.
// Nilai dari environment proses selalu menang, sehingga aman untuk production
// yang tidak memakai file .env.
func Load() Config {
	loadDotEnv()

	return Config{
		AppEnv:      getString("APP_ENV", "development"),
		Port:        getString("PORT", "8080"),
		DatabaseURL: getString("DATABASE_URL", "postgres://admin:admin@localhost:5432/simonjp?sslmode=disable"),
		DB: DBConfig{
			MaxOpenConns:    getInt("DB_MAX_OPEN_CONNS", 10),
			MaxIdleConns:    getInt("DB_MAX_IDLE_CONNS", 2),
			ConnMaxLifetime: getDuration("DB_CONN_MAX_LIFETIME", 5*time.Minute),
			ConnMaxIdleTime: getDuration("DB_CONN_MAX_IDLE_TIME", time.Minute),
		},
	}
}

func loadDotEnv() {
	// Tanpa .env bukan error: di production konfigurasi bisa datang dari
	// environment proses atau default di atas.
	if err := godotenv.Load(); err != nil && !errors.Is(err, os.ErrNotExist) {
		slog.Warn("gagal membaca file .env", "error", err)
	}
}

func getString(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok && value != "" {
		return value
	}
	return fallback
}

func getInt(key string, fallback int) int {
	raw := getString(key, "")
	if raw == "" {
		return fallback
	}

	value, err := strconv.Atoi(raw)
	if err != nil {
		slog.Warn("nilai konfigurasi bukan angka, memakai default", "key", key, "value", raw, "default", fallback)
		return fallback
	}
	return value
}

func getDuration(key string, fallback time.Duration) time.Duration {
	raw := getString(key, "")
	if raw == "" {
		return fallback
	}

	value, err := time.ParseDuration(raw)
	if err != nil {
		slog.Warn("nilai konfigurasi bukan durasi, memakai default", "key", key, "value", raw, "default", fallback)
		return fallback
	}
	return value
}
