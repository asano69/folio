MAKEOPTS := "-r"

# Tools
ESBUILD := esbuild

# Paths
SRC_DIR := src

# App
ENTRY := $(SRC_DIR)/main.ts
OUT   := static/app.js

# ─────────────────────────────────────────
# Default: build & run
# ─────────────────────────────────────────
.PHONY: all
all: $(OUT)
	go run ./cmd/folio/ server

# ─────────────────────────────────────────
# Build
# ─────────────────────────────────────────
$(OUT): $(shell find $(SRC_DIR) -type f)
	$(ESBUILD) $(ENTRY) \
		--bundle \
		--format=esm \
		--sourcemap \
		--outfile=$(OUT)

# ─────────────────────────────────────────
# Development
# ─────────────────────────────────────────
.PHONY: watch
watch:
	air

# ─────────────────────────────────────────
# Docker
# ─────────────────────────────────────────
.PHONY: docker-up
docker-up:
	docker compose up --build --force-recreate

.PHONY: docker-build
docker-build:
	docker build -t folio:latest .

# ─────────────────────────────────────────
# Clean
# ─────────────────────────────────────────
.PHONY: clean
clean:
	rm -f $(OUT)

# ─────────────────────────────────────────
# Help
# ─────────────────────────────────────────
.PHONY: help
help:
	@echo "Usage: make [target]"
