.PHONY: all test build clean

all: build test

build:
	@for d in ai agent tui coding-agent mom; do \
		echo "Building $$d..."; \
		(cd $$d && go build ./... || true); \
	done

test:
	@for d in ai agent tui coding-agent mom; do \
		echo "Testing $$d..."; \
		(cd $$d && go test ./...); \
	done

clean:
	@for d in ai agent tui coding-agent mom; do \
		echo "Cleaning $$d..."; \
		(cd $$d && go clean); \
	done
