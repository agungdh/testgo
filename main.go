package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"testgo/internal/config"
	"testgo/internal/db"
	"testgo/internal/repository"
	"testgo/internal/router"
)

func main() {
	cfg := config.Load()

	database := db.NewDB(cfg)
	defer database.Close()

	startupCtx, cancelStartup := context.WithTimeout(context.Background(), 30*time.Second)

	if err := database.PingContext(startupCtx); err != nil {
		slog.Error("gagal terhubung ke database", "error", err)
		os.Exit(1)
	}

	if cfg.DB.AutoMigrate {
		if err := db.RunMigrations(startupCtx, database.DB, "up"); err != nil {
			slog.Error("gagal menjalankan migrasi", "error", err)
			os.Exit(1)
		}
	}
	cancelStartup()

	repo := repository.NewPostgresProductRepository(database)

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           router.New(repo),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	// Jalankan server di goroutine agar bisa menangani graceful shutdown.
	serverErr := make(chan error, 1)
	go func() {
		slog.Info("server berjalan", "address", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	select {
	case err := <-serverErr:
		slog.Error("server gagal berjalan", "error", err)
		os.Exit(1)
	case <-ctx.Done():
		slog.Info("menerima sinyal shutdown, menutup server...")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("gagal shutdown dengan rapi", "error", err)
		os.Exit(1)
	}

	slog.Info("server berhenti")
}
