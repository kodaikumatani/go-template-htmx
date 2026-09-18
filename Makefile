.PHONY: run build test fmt vet check

run:
	go run ./cmd/server

build:
	go build -o bin/server ./cmd/server

test:
	go test ./...

fmt:
	gofmt -l -w .

vet:
	go vet ./...

check: vet test
