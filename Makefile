all: build

build:
	go build -o bin/ghx ./cmd/ghx
	go build -o bin/ghxd ./cmd/ghxd

test:
	go test ./...

clean:
	rm -rf bin/

.PHONY: all build test clean
