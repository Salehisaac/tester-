BINARY_NAME=tester
GO_FLAGS=CGO_ENABLED=0 GOOS=linux GOARCH=amd64

build:
	@CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o tester ./cmd/app
	@go run cmd/zip_maker.go

	