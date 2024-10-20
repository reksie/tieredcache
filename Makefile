.PHONY: all build run test start

all: build

clean:
	rm ./dist/*

build:clean
	go build -o dist/memo ./cmd/memo

run:
	go run ./cmd/memo

test:
	go test -v ./pkg/...

start:
	air
