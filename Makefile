BIN_DIR := bin
BINARY  := $(BIN_DIR)/sleeper

.PHONY: build test clean

build:
	go build -o $(BINARY) ./cmd/sleeper

test:
	go test ./...

clean:
	rm -rf $(BIN_DIR)
