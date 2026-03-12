.PHONY: run dev build build-debug test lint clean

BINARY     := ./bin/counter
BINARY_DBG := ./bin/counter-debug
MAIN       := ./cmd/counter

## run — production build and run
run: build
	$(BINARY)

## dev — hot-reload development mode (requires debug tag)
dev:
	@chmod +x scripts/dev.sh
	@./scripts/dev.sh

## build — production binary (no devtools)
build:
	@mkdir -p ./bin
	go build -o $(BINARY) $(MAIN)

## build-debug — debug binary with devtools + hot reload
build-debug:
	@mkdir -p ./bin
	go build -tags debug -o $(BINARY_DBG) $(MAIN)

## test — run all tests
test:
	go test ./...

## lint — run go vet + staticcheck
lint:
	go vet ./...
	staticcheck ./...

## clean — remove built binaries
clean:
	rm -rf ./bin
