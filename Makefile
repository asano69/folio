MAKEOPTS := "-r"

# Tools
TSC     := tsc
ESBUILD := esbuild

# Paths (override if needed)
JS_DIR  := .js
SRC_DIR := src

# App
APP := app.js
OUT := static/$(APP)

FLAG := $(JS_DIR)/.tsflag

# ─────────────────────────────────────────
# Default: build & run
# ─────────────────────────────────────────
.PHONY: all
all: $(OUT)
	go run cmd/server/main.go

# ─────────────────────────────────────────
# Build
# ─────────────────────────────────────────
$(OUT): $(FLAG)
	$(ESBUILD) --bundle $(JS_DIR)/$(APP) > $@

$(FLAG): $(shell find $(SRC_DIR) -type f)
	$(TSC) -p $(SRC_DIR)
	touch $@

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
	docker build -t openbook:latest .

# ─────────────────────────────────────────
# Clean
# ─────────────────────────────────────────
.PHONY: clean
clean:
	rm -rf $(JS_DIR)
	rm -f  $(OUT)

# ─────────────────────────────────────────
# Help
# ─────────────────────────────────────────
.PHONY: help
help:
	@echo "Usage: make [target]"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) \
		| sort \
		| awk 'BEGIN {FS=":.*?## "}; {printf "  %-15s %s\n", $$1, $$2}'
