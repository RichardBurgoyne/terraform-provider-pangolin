default: fmt lint install generate

build:
	go build -v ./...

install: build
	go install -v ./...

lint:
	golangci-lint run

generate:
	go generate ./...
	tfplugindocs generate

fmt:
	gofmt -s -w -e .

test:
	go test -v -cover -race ./...

.PHONY: fmt lint test build install generate
