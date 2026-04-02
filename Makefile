VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-X github.com/Sudhan30/freshprobe/internal/version.Version=$(VERSION)"
BIN := bin/freshprobe

.PHONY: build test lint vet clean docker cross

build:
	go build $(LDFLAGS) -o $(BIN) ./cmd/freshprobe

test:
	go test -race ./...

vet:
	go vet ./...

lint: vet
	@which golangci-lint > /dev/null 2>&1 && golangci-lint run || echo "golangci-lint not installed, skipping (go vet passed)"

clean:
	rm -rf bin/

docker:
	docker build -t sudhan03/freshprobe:$(VERSION) .

cross:
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o bin/freshprobe-linux-amd64 ./cmd/freshprobe
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o bin/freshprobe-darwin-arm64 ./cmd/freshprobe
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o bin/freshprobe-windows-amd64.exe ./cmd/freshprobe
