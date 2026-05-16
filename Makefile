APP_NAME := proxera-agent
PKG := github.com/wenisch-tech/proxera-agent
SRC_DIR := src
VERSION ?= dev
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
BUILD_DATE ?= $(shell powershell -NoProfile -Command "(Get-Date).ToUniversalTime().ToString('yyyy-MM-ddTHH:mm:ssZ')")

LDFLAGS := -X $(PKG)/internal/version.Version=$(VERSION) -X $(PKG)/internal/version.Commit=$(COMMIT) -X $(PKG)/internal/version.BuildDate=$(BUILD_DATE)

.PHONY: build test fmt tidy run

build:
	go -C $(SRC_DIR) build -ldflags "$(LDFLAGS)" -o ../bin/$(APP_NAME) ./cmd/$(APP_NAME)

test:
	go -C $(SRC_DIR) test ./...

fmt:
	gofmt -w ./src/cmd ./src/internal

tidy:
	go -C $(SRC_DIR) mod tidy

run:
	go -C $(SRC_DIR) run -ldflags "$(LDFLAGS)" ./cmd/$(APP_NAME)
