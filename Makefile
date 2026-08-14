BINARY := repogrep
CMD    := ./cmd/repogrep

.PHONY: build clean lint fmt test

build:
	go build -o $(BINARY) $(CMD)

clean:
	rm -f $(BINARY)
	go clean

lint:
	golangci-lint run ./...

fmt:
	gofmt -l -w .

test:
	go test ./...
