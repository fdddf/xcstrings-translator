BIN=xcstrings-translator

all: binary

binary:
	go build -o $(BIN)

clean:
	rm -f $(BIN)

test:
	go test -v

install:
	go install

PHONY: binary clean test install