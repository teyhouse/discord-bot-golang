.PHONY: test lint vuln run build

test:
	go test -race -shuffle=on ./...

lint:
	go vet ./...
	go tool staticcheck ./...

vuln:
	govulncheck ./...

run:
	go run ./cmd/bot

build:
	go build -o bot ./cmd/bot
