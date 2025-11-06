APP=node-module-man

.PHONY: build all test fmt fixtures version

build:
	./build.sh

all:
	./build.sh all

test:
	go test ./...

fmt:
	gofmt -w .

fixtures:
	node scripts/make-test-fixtures.js --count 4

# Usage: make version VERSION=1.2.3
version:
	VERSION=$(VERSION) ./build.sh
