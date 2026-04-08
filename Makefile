MAKEOPTS := "-r"
TSC      := tsc
ESBUILD  := esbuild
JS       := .js
FLAG     := $(JS)/.tsflag
APP      := my-mind.js
OUT      := static/$(APP)

# ─────────────────────────────────────────
#  Default: build & start server
# ─────────────────────────────────────────
.PHONY: all
all: $(OUT) ## (*) Bundle my-mind.js and start the server
	go run cmd/server/main.go

# ─────────────────────────────────────────
#  Build rules
# ─────────────────────────────────────────
$(OUT): $(FLAG)
	$(ESBUILD) --bundle $(JS)/$(APP) > $@

# toast.js is loaded as a plain ES module by catalog.html and others,
# so it is copied as-is rather than bundled.
$(TOAST_JS): $(FLAG)
	cp $(JS)/ui/toast.js $@

$(FLAG): $(shell find src -type f)
	$(TSC) -p src
	touch $@

# ─────────────────────────────────────────
#  Development
# ─────────────────────────────────────────
.PHONY: watch
watch: ## Watch for changes and reload (requires air)
	air

# ─────────────────────────────────────────
#  Docker / deploy
# ─────────────────────────────────────────
.PHONY: docker-up
docker-up: ## Build and start with Docker Compose
	docker compose -f docker-compose.dev.yaml up --build --force-recreate

.PHONY: docker-build
docker-build: ## Build Docker image
	docker build -t registry.internal/my-mind:latest .

.PHONY: docker-push
docker-push: ## Push Docker image
	docker push registry.internal/my-mind:latest

.PHONY: deploy
deploy: docker-build docker-push ## (*) Deploy stack via Komodo
	docker exec -it komodo km x -y destroy-stack my-mind
	docker exec -it komodo km x -y pull-stack   my-mind
	docker exec -it komodo km x -y deploy-stack my-mind

# ─────────────────────────────────────────
#  Misc
# ─────────────────────────────────────────
.PHONY: clean
clean: ## Remove build artifacts
	rm -rf $(JS)
	rm -f  $(OUT)
	rm -f  $(TOAST_JS)

.PHONY: help
help: ## Show available targets
	@echo "Usage: make [target]"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) \
		| sort \
		| awk 'BEGIN {FS=":.*?## "}; {printf "  %-15s %s\n", $$1, $$2}'
