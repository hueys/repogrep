BINARY  := repogrep
CMD     := ./cmd/repogrep
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -X main.version=$(VERSION)

.PHONY: build clean lint fmt test

build:
	go build -ldflags "$(LDFLAGS)" -o $(BINARY) $(CMD)

clean:
	rm -f $(BINARY)
	go clean

lint:
	golangci-lint run ./...
	zizmor .github/workflows/

fmt:
	gofmt -l -w .

test:
	go test ./...
