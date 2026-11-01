VERSION ?=
LDFLAGS = -s -w -X github.com/chyok/st/config.Version=$(VERSION)

build:
	CGO_ENABLED=0 go build -trimpath -ldflags "$(LDFLAGS)" -o st

test:
	go test -race ./...

lint:
	golangci-lint run

# Cross-compile every release target into dist/, as releases do.
snapshot:
	goreleaser build --snapshot --clean

clean:
	rm -rf st dist

.PHONY: build test lint snapshot clean
