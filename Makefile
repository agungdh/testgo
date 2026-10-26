APP_NAME  := testgo
BIN_DIR   := build
BIN       := $(BIN_DIR)/$(APP_NAME)

GO        ?= go
PORT      ?= 8080

IMAGE     ?= $(APP_NAME):latest
PLATFORM  ?= linux/amd64
# Level microarchitecture minimum: v2 (default), v1, v3, atau v4.
GOAMD64   ?= v2

# Build untuk production: statis, tanpa debug info, path dibersihkan.
PROD_GOOS   ?= linux
PROD_GOARCH ?= amd64
PROD_BIN    := $(BIN_DIR)/$(APP_NAME)-$(PROD_GOOS)-$(PROD_GOARCH)
LDFLAGS     := -s -w

.DEFAULT_GOAL := help

.PHONY: help
help: ## Tampilkan daftar perintah
	@echo "Penggunaan: make <target>"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) \
		| sort \
		| awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'

.PHONY: build
build: ## Build binary untuk lokal (development)
	@mkdir -p $(BIN_DIR)
	CGO_ENABLED=0 GOAMD64=$(GOAMD64) $(GO) build -trimpath -o $(BIN) .

.PHONY: build-prod
build-prod: ## Build binary production (statis, stripped, linux/amd64)
	@mkdir -p $(BIN_DIR)
	CGO_ENABLED=0 GOOS=$(PROD_GOOS) GOARCH=$(PROD_GOARCH) GOAMD64=$(GOAMD64) \
		$(GO) build -trimpath -ldflags="$(LDFLAGS)" -o $(PROD_BIN) .
	@echo "binary: $(PROD_BIN)"

.PHONY: dev
dev: ## Jalankan aplikasi untuk development (auto-reload jika air terpasang)
	@if command -v air >/dev/null 2>&1; then \
		echo "menjalankan air (auto-reload) di port $(PORT)"; \
		PORT=$(PORT) air; \
	else \
		echo "air tidak ditemukan, memakai 'go run' (tanpa auto-reload)"; \
		echo "install air: make install-air"; \
		PORT=$(PORT) $(GO) run .; \
	fi

.PHONY: run
run: build ## Jalankan binary hasil build
	PORT=$(PORT) ./$(BIN)

.PHONY: test
test: ## Jalankan semua test
	$(GO) test ./...

.PHONY: test-cover
test-cover: ## Jalankan test dengan laporan coverage
	$(GO) test -coverprofile=coverage.out ./...
	$(GO) tool cover -func=coverage.out

.PHONY: vet
vet: ## Analisis statis kode
	$(GO) vet ./...

.PHONY: fmt
fmt: ## Format kode
	$(GO) fmt ./...

.PHONY: tidy
tidy: ## Rapikan go.mod dan go.sum
	$(GO) mod tidy

.PHONY: check
check: fmt vet test ## Format, analisis statis, dan test

.PHONY: install-air
install-air: ## Install air untuk auto-reload saat development
	$(GO) install github.com/air-verse/air@latest

.PHONY: docker-build
docker-build: ## Build image Docker (multi-stage, distroless)
	docker build --platform $(PLATFORM) --build-arg GOAMD64=$(GOAMD64) -t $(IMAGE) .

.PHONY: docker-run
docker-run: ## Jalankan image Docker (host port dari PORT)
	docker run --rm -p $(PORT):8080 -e PORT=8080 $(IMAGE)

.PHONY: clean
clean: ## Hapus hasil build dan file coverage
	rm -rf $(BIN_DIR) coverage.out
