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

# Dev convenience: same production image, config injected from .env at run
# time so secrets never enter an image layer.
dev:
	docker build -t discord-bot .
	@docker rm discord-bot-dev > /dev/null 2>&1 || true
	docker run --rm --name discord-bot-dev --env-file .env discord-bot
