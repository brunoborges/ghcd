all: build

build:
	go build -o bin/ghc ./cmd/ghc
	go build -o bin/ghcd ./cmd/ghcd

test:
	go test ./...

clean:
	rm -rf bin/

.PHONY: all build test clean
