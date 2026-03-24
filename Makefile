VERSION ?= 0.1.0
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
LDFLAGS  = -ldflags "-s -w -X main.Version=$(VERSION) -X main.Commit=$(COMMIT)"

GO       = go
GOFLAGS  = -trimpath
BIN_DIR  = bin

.PHONY: all build-manager build-agent build lint format test clean dev proto-gen \
        dev-web build-web lint-web format-web test-web \
        build-all build-agent-dist docker-build docker-push release

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
	golangci-lint run ./...

format:
	gofmt -s -w .
	goimports -w -local github.com/onflux-tech/dockeeper .

test:
	$(GO) test -count=1 ./...

clean:
	rm -rf $(BIN_DIR) web/dist

dev:
	$(GO) run ./cmd/manager serve --http=0.0.0.0:8090 --dir=./dk_data

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
