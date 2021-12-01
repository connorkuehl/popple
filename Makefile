VERSION := v0.0.0+dev
BUILD := $(shell git describe --tags --dirty 2>/dev/null || echo "$(VERSION)")

LD_FLAGS := "-X 'github.com/connorkuehl/popple.Version=$(BUILD)'"

PREFIX ?= /usr/local
BINDIR ?= $(PREFIX)/bin

GOSRC := $(shell find . -type f -name '*.go' -o -name '*.sql')
GOSRC += go.mod go.sum

.PHONY: build clean install test

all: build

build: popple

popple: $(GOSRC)
	@go build -v -ldflags=$(LD_FLAGS) -o $@ ./cmd/discord/popple/

test:
	@go test -v -ldflags=$(LD_FLAGS) ./...

install: popple
	@mkdir -m 755 -p $(BINDIR)
	@install -m 755 popple $(BINDIR)/popple

uninstall:
	rm -f $(BINDIR)/popple

clean:
	@rm -f popple
	@go clean -r
