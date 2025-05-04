PKG := `go list -f {{.Dir}} ./...`

fmt:
	@golangci-lint fmt

lint:
	@golangci-lint version
	@#golangci-lint config verify
	@golangci-lint run

test:
	@go test -v ./...

build:
	@go build github.com/vmkteam/colgen/cmd/colgen

mod:
	@go mod tidy

install: build
	@cp colgen ~/go/bin

all: fmt lint build install
