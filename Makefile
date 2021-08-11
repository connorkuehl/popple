VERSION := v2.2.0+dev
BUILD := $(shell git describe --tags 2>/dev/null || echo "$(VERSION)")

LD_FLAGS := "-X 'main.Version=$(BUILD)'"

SOURCES := $(shell find . -type f -name '*.go')
SOURCES += go.mod go.sum

.PHONY: build clean test

all: popple

popple: build

build: $(SOURCES)
	@go build -v -ldflags=$(LD_FLAGS) ./...

test: popple
	@go test -v -ldflags=$(LD_FLAGS) ./...

clean:
	@rm -rf popple
