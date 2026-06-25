BINARY := murmur
PKG := github.com/Jaikin-SASU/murmuration
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo v0.1.0-dev)
LDFLAGS := -s -w -X $(PKG)/internal/version.Version=$(VERSION)

.PHONY: all build test vet cover fmt dist clean

all: vet test build

build:
	@mkdir -p bin
	go build -ldflags "$(LDFLAGS)" -o bin/$(BINARY) ./cmd/murmur

test:
	go test -race -cover ./...

vet:
	go vet ./...

cover:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Rapport : coverage.html"

fmt:
	gofmt -s -w .

# Binaires multi-OS pour le déploiement sur le parc.
dist:
	@mkdir -p dist
	GOOS=windows GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o dist/$(BINARY)-windows-amd64.exe ./cmd/murmur
	GOOS=darwin  GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o dist/$(BINARY)-darwin-arm64 ./cmd/murmur
	GOOS=darwin  GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o dist/$(BINARY)-darwin-amd64 ./cmd/murmur
	GOOS=linux   GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o dist/$(BINARY)-linux-amd64 ./cmd/murmur
	@echo "Binaires dans dist/"

clean:
	rm -rf bin dist coverage.out coverage.html
