.PHONY: build test lint fmt fmt-check clean

build:
	go build ./...

test:
	go test ./... -v -race

lint:
	golangci-lint run ./...

fmt:
	gofmt -w .
	goimports -w .

fmt-check:
	@tmp_dir="$$(mktemp -d)"; \
	trap 'rm -rf "$$tmp_dir"' EXIT; \
	rsync -a --exclude='.git/' ./ "$$tmp_dir"/; \
	(cd "$$tmp_dir" && gofmt -w . && goimports -w .); \
	diff -ruN --exclude=.git . "$$tmp_dir"

clean:
	go clean ./...
