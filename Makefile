APP_NAME := testgo
BIN_DIR  := build
BIN      := $(BIN_DIR)/$(APP_NAME)

MIGRATION_DIR := internal/db/migrations
GOOSE_VERSION := v3.29.0

GO ?= go

.DEFAULT_GOAL := help

.PHONY: help
help: ## Tampilkan daftar perintah
	@echo "Penggunaan: make <target>"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) \
		| sort \
		| awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-16s\033[0m %s\n", $$1, $$2}'

.PHONY: env
env: ## Salin .env.example ke .env jika belum ada
	@test -f .env || cp .env.example .env
	@echo "Konfigurasi siap di .env"

.PHONY: build
build: ## Build binary untuk development
	@mkdir -p $(BIN_DIR)
	$(GO) build -o $(BIN) .

.PHONY: dev
dev: env ## Jalankan aplikasi untuk development
	$(GO) run .

.PHONY: run
run: build ## Jalankan binary hasil build
	./$(BIN)

.PHONY: test
test: ## Jalankan semua test
	$(GO) test ./...

.PHONY: migrate-up
migrate-up: env ## Terapkan semua migrasi database
	$(GO) run ./cmd/migrate up

.PHONY: migrate-down
migrate-down: env ## Rollback satu migrasi terakhir
	$(GO) run ./cmd/migrate down

.PHONY: migrate-status
migrate-status: env ## Tampilkan status migrasi
	$(GO) run ./cmd/migrate status

.PHONY: migrate-version
migrate-version: env ## Tampilkan versi migrasi aktif
	$(GO) run ./cmd/migrate version

.PHONY: migrate-create
migrate-create: ## Buat migrasi baru: make migrate-create name=nama_migrasi
	@test -n "$(name)" || (echo "gunakan: make migrate-create name=nama_migrasi" && exit 1)
	$(GO) run github.com/pressly/goose/v3/cmd/goose@$(GOOSE_VERSION) -dir $(MIGRATION_DIR) create $(name) sql

.PHONY: clean
clean: ## Hapus hasil build
	rm -rf $(BIN_DIR)
