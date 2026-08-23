.PHONY: test lint vuln run build dev

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

# Dev-only: builds the local .env into the image; never push it.
dev:
	docker build -f Dockerfile-dev -t discord-bot:dev .
	@docker rm discord-bot-dev > /dev/null 2>&1 || true
	docker run --rm --name discord-bot-dev discord-bot:dev
