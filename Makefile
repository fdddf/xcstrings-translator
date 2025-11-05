BIN=xcstrings-translator

VERSION=$(shell git describe --tags --abbrev=0 2>/dev/null || echo "dev")
DATE=$(shell date +%Y%m%d%H%M%S)
COMMIT:=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

LDFLAGS=-s -w
LDFLAGS+= -X github.com/fdddf/xcstrings-translator/cmd.Version=$(VERSION)
LDFLAGS+= -X github.com/fdddf/xcstrings-translator/cmd.Commit=$(COMMIT)
LDFLAGS+= -X github.com/fdddf/xcstrings-translator/cmd.Build=$(DATE)

all: binary

binary:
	go build -v -ldflags="$(LDFLAGS)" -o $(BIN) main.go

clean:
	rm -f $(BIN)

test:
	go test -v

install:
	go install -ldflags="$(LDFLAGS)"

PHONY: binary clean test install