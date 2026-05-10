all: build

.PHONY: build
build:
	go build

.PHONY: build-static
build-static:
	CGO_ENABLED=0 GOOS=linux \
		    go build -ldflags="-extldflags=-static" -tags netgo

.PHONY: test
test:
	mise run test

.PHONY: lint
lint:
	mise run lint
