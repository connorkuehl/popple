VERSION := v0.0.0+dev
BUILD := $(shell git describe --tags --dirty 2>/dev/null || echo "$(VERSION)")

LD_FLAGS := "-X 'github.com/connorkuehl/popple.Version=$(BUILD)'"

.PHONY: build clean lib test

all: build

build: lib ./cmd/discord/popple/popple

./cmd/discord/popple/popple:
	@echo "==> building $@"
	@go build -v -ldflags=$(LD_FLAGS) -o $@ ./cmd/discord/popple/

lib:
	@echo "==> checking popple"
	@go build -v -ldflags=$(LD_FLAGS)

test:
	@go test -v -ldflags=$(LD_FLAGS) ./...

clean:
	@rm -rf ./cmd/discord/popple/popple
	@go clean -r
