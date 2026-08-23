# discord-bot-golang

A modular Discord bot template written in Go, built on [bwmarrin/discordgo](https://github.com/bwmarrin/discordgo). It ships with two slash commands (ping, whisper), a reaction watcher that replies to recent messages carrying an eyes reaction, and a mention handler that summarizes messages the bot is tagged in.

## Features

- Slash commands registered guild-scoped on startup for immediate availability: public `ping`, whitelisted `whisper` and `pn` (sends the invoker a direct message), and whitelisted `summary` that lists the last 20 channel messages (bot messages and content-less ones excluded, each cut to 15 characters), answered ephemerally
- Optional whitelist permission middleware for slash commands
- Reaction watcher: scans the configured channel every 30 seconds and replies "I saw that!" to messages from the last 5 minutes that have an eyes reaction. Processed message IDs are persisted to daily JSON files under `data/` so duplicates are also suppressed across restarts. Files older than two days are pruned automatically.
- Mention handler: replies to direct @mentions of the bot with a truncated, markdown-stripped summary of the message
- Structured JSON logging via `log/slog`, level configurable through `LOG_LEVEL`
- pprof endpoint on `127.0.0.1:6060` for local profiling
- Graceful shutdown on SIGINT/SIGTERM

## How it works

The entry point in `cmd/bot/main.go` is wiring only: it loads configuration, builds the logger, creates the Discord client, registers commands and handlers through the router, starts the reaction watcher goroutine, then blocks until the context is cancelled by an OS signal.

All Discord logic lives under `internal/`. The client owns session lifecycle and intents. The router maps command names to handlers and dispatches interactions. Command handlers, event handlers, and the permission middleware are separate packages, so new commands or handlers are added by extending their package and registering them in the router without touching the client.

Configuration comes from the environment or a `.env` file at the project root. See `.env.example` for the available variables: `BOT_TOKEN`, `CHANNEL_ID` and `GUILD_ID` are required, `WHITELISTED_USER_IDS` and `LOG_LEVEL` are optional. Missing required values abort startup.

The reaction watcher uses a single worker on a ticker rather than concurrent scans, which keeps API usage predictable and well inside rate limits. It fetches up to 100 recent messages per tick, filters by timestamp, checks reactions for the Unicode emoji U+1F440, and skips any message already present in the seen store.

## Permission middleware

Slash command access is controlled by a composable middleware in `internal/discord/middleware`. `WhitelistChecker` holds a set of user IDs parsed from `WHITELISTED_USER_IDS` (comma separated). `WithPermissionCheck` wraps a handler and rejects invocations from users not in that set with an ephemeral "Not authorized" response.

An empty `WHITELISTED_USER_IDS` disables the check entirely: `Allowed` returns true for everyone, making the bot public. No code changes are needed to toggle between modes.

To exempt a single command from the check while keeping it for others, call `router.RegisterPublicCommand("ping")` before `RegisterHandlers()`. To replace whitelisting altogether, implement the small `PermissionChecker` interface (`Allowed(userID string) bool`) and pass your implementation to `discord.NewRouter`.

## Building and running locally

Requirements:

- Go 1.24 or newer (the toolchain is pinned via `go.mod`; staticcheck runs through the `tool` directive, no separate install needed)
- govulncheck on your PATH for `make vuln`: `go install golang.org/x/vuln/cmd/govulncheck@latest`

```sh
cp .env.example .env   # fill in token, channel and guild IDs
make run               # go run ./cmd/bot
make build             # binary at bin/bot
```

Common tasks:

| Command | What it does |
| --- | --- |
| `make test` | Tests with race detector and shuffled order |
| `make lint` | `go vet` plus staticcheck |
| `make vuln` | govulncheck against reachable vulnerabilities |
| `make run` | Run the bot directly |
| `make build` | Build the `bot` binary in the project root |

## Docker

The Dockerfile builds a multi-stage image: a golang builder produces a fully static binary (`CGO_ENABLED=0`, stripped symbols, trimpath), and the runtime stage is `FROM scratch`. The resulting image contains only the binary, a CA certificate bundle for Discord's TLS endpoints, and nothing else: no shell, no libc, no package manager. It runs as UID 65532, non-root.

Configuration is passed through environment variables (the config loader reads them natively; no `.env` needed in the container). Mount a directory at `/data` so reaction watcher state survives restarts:

```sh
docker build -t discord-bot .
docker run --rm \
  -e BOT_TOKEN=... -e CHANNEL_ID=... -e GUILD_ID=... \
  -v ./data:/data \
  discord-bot
```

The state directory defaults to `data` relative to the working directory and can be moved with the `DATA_DIR` environment variable in any deployment.


Discord prerequisites: create an application and bot account in the developer portal, enable the Message Content intent, invite it with the appropriate scopes, and grab channel/guild/user IDs via Developer Mode's Copy ID.

## Project layout

Domain packages own their types and logic; there are no utils or common packages. `internal/` enforces privacy at compile time. Commands live in `internal/discord/commands`, event handlers in `internal/discord/handlers`, permission middleware in `internal/discord/middleware`, and routing plus session management in `internal/discord`. Configuration loading and logger construction sit in their own small packages.

## Using this project

Treat it as a template, not a library. Everything lives under `internal/`, which the Go compiler restricts to imports from within this module only, so no package here can be imported by another repository. That is deliberate: a bot is an application, and its Discord wiring is specific to its deployment.

The intended workflow is to copy the project (or GitHub's "Use this template") and rename the module in `go.mod`. From there you add commands to `commands`, handlers to `handlers`, and register them in the router. Configuration, logging, middleware, and shutdown handling come for free.

If you later find genuinely reusable pieces, such as the whitelist middleware or the seen store, promote them out of `internal/` into their own module at that point. Extract on second use rather than designing for reuse up front.

