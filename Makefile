.PHONY: build test lint fmt clean

build:
	go build ./...

test:
	go test ./... -v -race

lint:
	golangci-lint run ./...

fmt:
	gofmt -w .
	goimports -w .

clean:
	go clean ./...
