// Command migrate menjalankan migrasi database memakai goose.
//
// Pemakaian:
//
//	go run ./cmd/migrate up
//	go run ./cmd/migrate down
//	go run ./cmd/migrate status
//	go run ./cmd/migrate version
//
// Migrasi yang sama otomatis dijalankan saat aplikasi start bila
// DB_AUTO_MIGRATE bernilai true.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"testgo/internal/config"
	"testgo/internal/db"
)

const usage = `penggunaan: migrate <perintah>

perintah:
  up       jalankan semua migrasi yang belum diterapkan
  down     rollback satu migrasi terakhir
  status   tampilkan status setiap migrasi
  version  tampilkan versi migrasi yang aktif
`

func main() {
	if len(os.Args) != 2 {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(2)
	}

	command := os.Args[1]

	cfg := config.Load()
	database := db.NewDB(cfg)
	defer database.Close()

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	if err := database.PingContext(ctx); err != nil {
		slog.Error("gagal terhubung ke database", "error", err)
		os.Exit(1)
	}

	if err := db.RunMigrations(ctx, database.DB, command); err != nil {
		slog.Error("migrasi gagal", "command", command, "error", err)
		os.Exit(1)
	}
}
