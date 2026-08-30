CGO_ENABLED ?= 1
CGO_CFLAGS  ?= -O2 -g0
GO         ?= go
BIN_DIR    := bin
BIN        := $(BIN_DIR)/ms2pdf
VERSION    ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
GOFLAGS    := -trimpath -buildvcs=false
LDFLAGS    := -s -w -X main.version=$(VERSION)

UNAME_S := $(shell uname -s 2>/dev/null)
ifeq ($(OS),Windows_NT)
STRIP :=
else ifeq ($(UNAME_S),Darwin)
STRIP := strip -x
else
STRIP := strip -s
endif

.PHONY: all build test vet fmt clean

all: build

build:
	CGO_ENABLED=$(CGO_ENABLED) CGO_CFLAGS="$(CGO_CFLAGS)" \
		$(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(BIN) ./cmd/ms2pdf
ifneq ($(STRIP),)
	$(STRIP) $(BIN)
endif

test:
	CGO_ENABLED=$(CGO_ENABLED) $(GO) test ./...

vet:
	CGO_ENABLED=$(CGO_ENABLED) $(GO) vet ./...

fmt:
	$(GO) fmt ./...

clean:
	rm -rf $(BIN_DIR)
