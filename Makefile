APP_NAME := testgo
BIN_DIR  := build
BIN      := $(BIN_DIR)/$(APP_NAME)

GO   ?= go
PORT ?= 8080

.DEFAULT_GOAL := help

.PHONY: help
help: ## Tampilkan daftar perintah
	@echo "Penggunaan: make <target>"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) \
		| sort \
		| awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'

.PHONY: build
build: ## Build binary untuk development
	@mkdir -p $(BIN_DIR)
	$(GO) build -o $(BIN) .

.PHONY: dev
dev: ## Jalankan aplikasi untuk development
	PORT=$(PORT) $(GO) run .

.PHONY: run
run: build ## Jalankan binary hasil build
	PORT=$(PORT) ./$(BIN)

.PHONY: test
test: ## Jalankan semua test
	$(GO) test ./...

.PHONY: clean
clean: ## Hapus hasil build
	rm -rf $(BIN_DIR)