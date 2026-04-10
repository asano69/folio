MAKEOPTS := "-r"

# Tools
ESBUILD := esbuild

# Paths
SRC_DIR := src

# JavaScript
JS_ENTRY  := $(SRC_DIR)/main.ts
JS_OUT    := static/app.js

# CSS
CSS_ENTRY := $(SRC_DIR)/style.css
CSS_OUT   := static/style.css

# ─────────────────────────────────────────
# Default: build & run
# ─────────────────────────────────────────
.PHONY: all
all: $(JS_OUT) $(CSS_OUT)
	go run ./cmd/folio/ server

# ─────────────────────────────────────────
# Build
# ─────────────────────────────────────────
$(JS_OUT): $(shell find $(SRC_DIR) -name "*.ts")
	$(ESBUILD) $(JS_ENTRY) \
		--bundle \
		--format=esm \
		--sourcemap \
		--outfile=$(JS_OUT)

$(CSS_OUT): $(shell find $(SRC_DIR) -name "*.css")
	$(ESBUILD) $(CSS_ENTRY) \
		--bundle \
		--outfile=$(CSS_OUT)

.PHONY: build
build:
	go build ./cmd/folio

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
# database
# ─────────────────────────────────────────
.PHONY: db
reset-db:  ## (*) Deploy stack via Komodo
	rm -f data/folio.db
	rm -f data/folio.db-shm
	rm -f data/folio.db-wal
	docker exec -it komodo km x -y destroy-stack dbgate
	#docker exec -it komodo km x -y pull-stack   my-mind
	docker exec -it komodo km x -y deploy-stack dbgate


# ─────────────────────────────────────────
# icon
# ─────────────────────────────────────────
.PHONY: icon
icon:
	magick -background none src/folio.svg \
	  \( -clone 0 -resize 16x16 \) \
	  \( -clone 0 -resize 32x32 \) \
	  \( -clone 0 -resize 48x48 \) \
	  -delete 0 static/favicon.ico

# ─────────────────────────────────────────
# Clean
# ─────────────────────────────────────────
.PHONY: clean
clean:
	rm -f $(JS_OUT) $(CSS_OUT)
	rm -f folio

# ─────────────────────────────────────────
# Help
# ─────────────────────────────────────────
.PHONY: help
help:
	@echo "Usage: make [target]"
