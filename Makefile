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
	go run ./cmd/folio/ serve

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
		--loader:.svg=dataurl \
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


.PHONY: typecheck
typecheck:
	cd src && tsc --noEmit

# ─────────────────────────────────────────
# Docker
# ─────────────────────────────────────────
.PHONY: build-container
build-container:
	docker compose -f compose.yaml up -d --build --force-recreate

# 開発中は、Komodoを使う必要はない
#.PHONY: build-image
#build-image: ## Build Docker image
#	docker build -t registry.internal/folio:latest .
#
#.PHONY: push-image
#push-image: ## Push Docker image
#	docker push registry.internal/folio:latest
#
#.PHONY: deploy
#deploy: build-image push-image ## (*) Deploy stack via Komodo
#	docker exec -it komodo km x -y destroy-stack folio
#	docker exec -it komodo km x -y pull-stack   folio
#	docker exec -it komodo km x -y deploy-stack folio


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
