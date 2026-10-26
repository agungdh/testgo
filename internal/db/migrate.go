package db

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"

	"github.com/pressly/goose/v3"
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

// RunMigrations menjalankan perintah goose (up, down, status, version)
// terhadap file migrasi yang di-embed ke dalam binary.
func RunMigrations(ctx context.Context, sqldb *sql.DB, command string) error {
	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("set dialek goose: %w", err)
	}

	migrations, err := fs.Sub(migrationFiles, "migrations")
	if err != nil {
		return fmt.Errorf("buka folder migrasi: %w", err)
	}
	goose.SetBaseFS(migrations)

	switch command {
	case "up":
		return goose.UpContext(ctx, sqldb, ".")
	case "down":
		return goose.DownContext(ctx, sqldb, ".")
	case "status":
		return goose.StatusContext(ctx, sqldb, ".")
	case "version":
		return goose.VersionContext(ctx, sqldb, ".")
	default:
		return fmt.Errorf("perintah migrasi tidak dikenal: %q", command)
	}
}
