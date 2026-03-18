VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
BINARY  := minicloud
LDFLAGS := -s -w -X main.version=$(VERSION)

# Default: build for current platform.
.PHONY: build
build:
	CGO_ENABLED=0 go build -trimpath -ldflags="$(LDFLAGS)" -o $(BINARY) ./cmd/minicloud

.PHONY: test
test:
	go test -count=1 -race ./...

.PHONY: lint
lint:
	@command -v golangci-lint >/dev/null 2>&1 || { echo "Install golangci-lint: https://golangci-lint.run/install/"; exit 1; }
	golangci-lint run ./...

# Docker
.PHONY: docker-build
docker-build:
	docker build --build-arg VERSION=$(VERSION) -t minicloud:$(VERSION) -t minicloud:latest .

.PHONY: docker-up
docker-up:
	VERSION=$(VERSION) docker compose up -d --build

.PHONY: docker-down
docker-down:
	docker compose down

# Cross-compile release binaries into dist/.
PLATFORMS := linux/amd64 linux/arm64 darwin/amd64 darwin/arm64
.PHONY: release
release:
	@mkdir -p dist
	@for platform in $(PLATFORMS); do \
		os=$${platform%%/*}; \
		arch=$${platform##*/}; \
		out=dist/$(BINARY)-$${os}-$${arch}; \
		[ "$$os" = "windows" ] && out=$${out}.exe; \
		echo "Building $${os}/$${arch} -> $${out}"; \
		GOOS=$${os} GOARCH=$${arch} CGO_ENABLED=0 \
			go build -trimpath -ldflags="$(LDFLAGS)" -o $${out} ./cmd/minicloud; \
	done
	@echo "Release binaries in dist/"

.PHONY: clean
clean:
	rm -f $(BINARY)
	rm -rf dist/

.PHONY: help
help:
	@echo "Targets:"
	@echo "  build        Build binary for current platform"
	@echo "  test         Run all tests with race detector"
	@echo "  lint         Run golangci-lint"
	@echo "  docker-build Build Docker image"
	@echo "  docker-up    Start with docker compose"
	@echo "  docker-down  Stop docker compose"
	@echo "  release      Cross-compile for linux/darwin amd64/arm64"
	@echo "  clean        Remove build artifacts"
