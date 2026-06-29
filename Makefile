# go-cli-template — shared build conventions for the m-cli Go toolchain.
# Every toolchain Go repo inherits this: static (CGO_ENABLED=0), -trimpath,
# version stamped via -ldflags, cross-compile matrix, lint, test, schema.

BIN     ?= hello                       # demo binary name (rename per repo)
PKG     := github.com/vista-cloud-dev/go-cli-template
# version vars (Version/Commit/Date) live in the imported clikit module
LDPKG   := github.com/vista-cloud-dev/clikit
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
DATE    ?= $(shell date -u +%Y-%m-%d)
LDFLAGS := -s -w -X $(LDPKG).Version=$(VERSION) -X $(LDPKG).Commit=$(COMMIT) -X $(LDPKG).Date=$(DATE)

# Static, no-libc, reproducible (spec §10).
GOFLAGS := -trimpath
export CGO_ENABLED := 0

PLATFORMS := linux/amd64 linux/arm64 darwin/arm64 windows/amd64

.PHONY: all build run lint test tidy schema dist clean

all: lint test build

build:
	go build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o dist/$(BIN) .

run: build
	./dist/$(BIN) $(ARGS)

lint:
	golangci-lint run ./...

# The race detector needs CGO; the rest of the build is CGO-free.
# Override the file-level CGO_ENABLED=0 just here.
#
# Coverage is REPORTED (-cover prints per-package %), not GATED, for Go repos —
# a deliberate org decision (org-conventions-baseline F6): the Go tools here are
# thin CLIs over the m/v engine seam where the meaningful coverage lives in the M
# suites (gated 80–85% via `m coverage`). A Go floor would gate mostly-trivial
# main/flag plumbing. Revisit per-repo if a Go module grows substantial own logic.
test:
	CGO_ENABLED=1 go test $(GOFLAGS) -race -cover ./...

tidy:
	go mod tidy

# Emit the machine schema (the §5.5 contract) — also a CI conformance artifact.
schema: build
	./dist/$(BIN) schema

# Cross-compile the pinned matrix into dist/.
dist:
	@mkdir -p dist
	@for p in $(PLATFORMS); do \
		os=$${p%/*}; arch=$${p#*/}; ext=""; [ "$$os" = "windows" ] && ext=".exe"; \
		echo "  $$os/$$arch"; \
		GOOS=$$os GOARCH=$$arch go build $(GOFLAGS) -ldflags "$(LDFLAGS)" \
			-o dist/$(BIN)-$$os-$$arch$$ext . ; \
	done

clean:
	rm -rf dist
