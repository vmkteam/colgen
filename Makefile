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
	@go build cmd/colgen/colgen.go

mod:
	@go mod tidy

install: build
	@cp colgen ~/go/bin