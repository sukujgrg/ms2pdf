CGO_ENABLED ?= 1
GO         ?= go
BIN_DIR    := bin
BIN        := $(BIN_DIR)/ms2pdf
VERSION    ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS    := -X main.version=$(VERSION)

.PHONY: all build test vet fmt clean

all: build

build:
	CGO_ENABLED=$(CGO_ENABLED) $(GO) build -ldflags "$(LDFLAGS)" -o $(BIN) ./cmd/ms2pdf

test:
	CGO_ENABLED=$(CGO_ENABLED) $(GO) test ./...

vet:
	CGO_ENABLED=$(CGO_ENABLED) $(GO) vet ./...

fmt:
	$(GO) fmt ./...

clean:
	rm -rf $(BIN_DIR)
