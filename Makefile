VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
LDFLAGS  = -ldflags "-s -w -X main.Version=$(VERSION) -X main.Commit=$(COMMIT)"

GO       = go
GOFLAGS  = -trimpath
BIN_DIR  = bin
GO_PACKAGES = ./cmd/... ./internal/... ./migrations/...

.PHONY: all build-manager build-agent build lint format test clean dev dev-manager dev-agent dev-all run proto-gen \
        dev-web build-web lint-web format-web test-web \
        build-all build-agent-dist docker-build docker-push release verify verify-fast pre-push hooks tools \
        harness-sync harness-check harness-test changelog-draft changelog-check release-check

all: build

## ── Go ───────────────────────────────────────────────

build: build-manager build-agent

build-manager:
	$(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BIN_DIR)/dockeeper-manager ./cmd/manager

build-agent:
	$(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BIN_DIR)/dockeeper-agent ./cmd/agent

## Cross-compile agent for linux/amd64 and linux/arm64 (dev testing for download endpoint)
build-agent-dist:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BIN_DIR)/dockeeper-agent-linux-amd64 ./cmd/agent
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 $(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BIN_DIR)/dockeeper-agent-linux-arm64 ./cmd/agent

## Build frontend then manager (production: embeds React into Go binary)
build-all: build-web build-manager build-agent

lint:
	golangci-lint run $(GO_PACKAGES)

format:
	gofmt -s -w cmd internal migrations
	goimports -w -local github.com/onflux-tech/dockeeper cmd internal migrations

tools:
	$(GO) install golang.org/x/tools/cmd/goimports@v0.42.0

harness-sync:
	node scripts/harness.mjs sync

harness-check:
	node scripts/harness.mjs check

harness-test:
	node --test scripts/harness.test.mjs scripts/changelog.test.mjs

changelog-draft:
	node scripts/changelog.mjs draft

changelog-check:
	node scripts/changelog.mjs check

release-check:
	node scripts/changelog.mjs check --version "$(VERSION)"

format-check:
	@files="$$(gofmt -l cmd internal migrations)"; if [ -n "$$files" ]; then echo "gofmt needed:"; echo "$$files"; exit 1; fi

test:
	$(GO) test -count=1 $(GO_PACKAGES)

clean:
	rm -rf $(BIN_DIR) tmp web/dist

dev:
	$(GO) run ./cmd/manager serve --http=0.0.0.0:8090 --dir=./dk_data

dev-manager:
	air -c .air.manager.toml

dev-agent:
	air -c .air.agent.toml

dev-all:
	$(MAKE) -j3 dev-manager dev-agent dev-web

run: dev-all

## ── Proto ────────────────────────────────────────────

proto-gen:
	@echo "Generating protobuf code..."
	cd proto && buf generate

## ── Web (React) ──────────────────────────────────────

dev-web:
	cd web && npm run dev

build-web:
	cd web && npm run build

lint-web:
	cd web && npm run lint

format-web:
	cd web && npm run format:check

test-web:
	cd web && npm test

## ── Verificação ──────────────────────────────────────

verify: format-check harness-check harness-test changelog-check build-web lint test lint-web format-web test-web

verify-fast: format-check harness-check changelog-check build-web test lint-web

pre-push: verify

## ── Hooks ───────────────────────────────────────────

hooks:
	git config core.hooksPath .githooks
	chmod +x .githooks/pre-push

## ── Docker ──────────────────────────────────────────

DOCKER_REPO ?= ghcr.io/onflux-tech

PLATFORMS ?= linux/amd64,linux/arm64

docker-build:
	docker buildx build -f Dockerfile.manager \
		--platform $(PLATFORMS) \
		--build-arg VERSION=$(VERSION) --build-arg COMMIT=$(COMMIT) \
		-t $(DOCKER_REPO)/dockeeper-manager:$(VERSION) \
		-t $(DOCKER_REPO)/dockeeper-manager:latest .
	docker buildx build -f Dockerfile.agent \
		--platform $(PLATFORMS) \
		--build-arg VERSION=$(VERSION) --build-arg COMMIT=$(COMMIT) \
		-t $(DOCKER_REPO)/dockeeper-agent:$(VERSION) \
		-t $(DOCKER_REPO)/dockeeper-agent:latest .

docker-push:
	docker buildx build -f Dockerfile.manager \
		--platform $(PLATFORMS) \
		--build-arg VERSION=$(VERSION) --build-arg COMMIT=$(COMMIT) \
		-t $(DOCKER_REPO)/dockeeper-manager:$(VERSION) \
		-t $(DOCKER_REPO)/dockeeper-manager:latest --push .
	docker buildx build -f Dockerfile.agent \
		--platform $(PLATFORMS) \
		--build-arg VERSION=$(VERSION) --build-arg COMMIT=$(COMMIT) \
		-t $(DOCKER_REPO)/dockeeper-agent:$(VERSION) \
		-t $(DOCKER_REPO)/dockeeper-agent:latest --push .

## ── Release ─────────────────────────────────────────

release:
	goreleaser release --clean
