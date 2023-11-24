all: build

.PHONY: build
build:
	go build

.PHONY: build-static
build-static:
	CGO_ENABLED=0 GOOS=linux \
		    go build -ldflags="-extldflags=-static" -tags netgo
