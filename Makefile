.PHONY: dev run build fmt vet

# ファイルを保存すると自動で再起動する開発用サーバ
# air が未インストールなら: go install github.com/air-verse/air@latest
dev:
	air

run:
	go run ./cmd/server

build:
	go build -o bin/server ./cmd/server

fmt:
	gofmt -l -w .

vet:
	go vet ./...
