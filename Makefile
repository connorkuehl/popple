VERSION := v0.0.0+dev
BUILD := $(shell git describe --tags --dirty 2>/dev/null || echo "$(VERSION)")

LD_FLAGS := "-X 'github.com/connorkuehl/popple.Version=$(BUILD)'"

SOURCES := $(shell find . -type f -name '*.go')
SOURCES += go.mod go.sum

.PHONY: clean test

all: ./cmd/discord/popple/popple

./cmd/discord/popple/popple: $(SOURCES)
	@go build -v -ldflags=$(LD_FLAGS) -o $@ ./cmd/discord/popple/

test:
	@go test -v -ldflags=$(LD_FLAGS) ./...

clean:
	@rm -rf ./cmd/discord/popple/popple
