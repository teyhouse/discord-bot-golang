# Architecture

This document describes how the bot is structured and why. The README covers
usage; this covers design.

## Overview

A modular Discord bot template built with
[bwmarrin/discordgo](https://github.com/bwmarrin/discordgo). The design
emphasizes separation of concerns, composable middleware, and minimal coupling
between Discord wiring and behavior. Everything lives under `internal/`, so no
package here can be imported from another module: this is an application, not
a library.

**Module**: `github.com/teyhouse/discord-bot-golang`
**Go**: 1.26+ (see `go.mod`; the Docker image tracks the same minor version)

## Layout

```
discord-bot-golang/
├── cmd/bot/main.go            # Entry point, wiring only
├── internal/
│   ├── config/                # Environment loading + validation
│   ├── discord/
│   │   ├── client.go          # Session lifecycle and intents
│   │   ├── router.go          # Command/event registration + dispatch
│   │   ├── commands/          # Slash command handlers (Registry type)
│   │   ├── handlers/          # Event handlers: mention responder,
│   │   │                      # :eyes: reaction watcher, seen store
│   │   └── middleware/        # Permission middleware (whitelist)
│   └── logger/                # Structured JSON logging via log/slog
└── docs/ARCHITECTURE.md       # This file
```

### Package organization principles

- **Domain over layers**: each package owns its types and logic (`commands/`,
  `handlers/`, `middleware/`).
- **No `utils/` or `common/`**: packages are named for what they provide.
- **Interfaces with the consumer**: small (1–3 methods), introduced when a
  second implementation actually appears.
- **Privacy via `internal/`**: compiler-enforced boundaries.

## Configuration

Loaded by `internal/config`: reads `.env` if present (real environment takes
precedence), validates required fields, and fails fast on missing values.

| Variable | Required | Meaning |
| --- | --- | --- |
| `BOT_TOKEN` | yes | Bot token from the developer portal |
| `CHANNEL_ID` | yes | Channel watched by the reaction watcher |
| `GUILD_ID` | yes | Guild for slash-command registration |
| `WHITELISTED_USER_IDS` | no | Comma-separated user IDs; empty = public bot |
| `LOG_LEVEL` | no | `debug`, `info` (default), `warn`, `error` |
| `DATA_DIR` | no | Seen-state directory (default `data`) |

## Discord client

`internal/discord.Client` owns the session lifecycle:

- Builds the session with the intents the handlers need: guild messages,
  guild message reactions, direct messages, message content.
- Does not connect in the constructor; `Start(ctx)` opens the gateway and
  blocks until the context is cancelled, then closes cleanly.
- Resolves the application ID over REST once (cached) so commands can be
  registered before or after connecting.

## Router

`internal/discord.Router` is the single registration point. It maps command
names to handlers from `commands.Registry`, registers all guild-scoped slash
commands on startup (immediate availability, no global propagation delay),
wraps non-public handlers with the permission middleware, and exposes
`RegisterPublicCommand` to exempt individual commands.

## Permission middleware

`internal/discord/middleware` defines a one-method interface:

```go
type PermissionChecker interface {
    Allowed(userID string) bool
}
```

`WhitelistChecker` implements it with an O(1) set of user IDs. An empty
whitelist makes `Allowed` return true for everyone — public mode without code
changes. `WithPermissionCheck` wraps any handler, denies non-whitelisted users
with an ephemeral response, and logs every decision at debug level.

To swap in role-based or DB-backed permissions, implement the interface and
pass it to `discord.NewRouter`.

## Slash commands

Handlers live on the `commands.Registry` value, which carries an injected
`slog.Logger`. Instant responses go through `middleware.Respond`. Commands
that do REST work before replying (`/summary`, `/pn`) acknowledge the
interaction with an ephemeral deferred response first — Discord only allows
3 seconds for direct responses, and followups after deferral have no such
limit.

Current commands: `ping` (public), `whisper`, `summary`, `pn` (whitelisted).

## Event handlers

- **Mention handler** (`handlers.MentionHandler`): replies to direct @mentions
  of the bot with a markdown-stripped summary of the message, truncated to
  200 characters. Mention markup is rewritten to readable `@usernames`.
- **Reaction watcher** (`handlers.ReactionWatcher`): a single worker on a
  30-second ticker scans up to 100 recent messages in the configured channel,
  filters to the last 5 minutes, and quotes messages carrying the 👀 emoji
  (U+1F440). Single-worker-by-design keeps API usage predictable and well
  inside rate limits.
- **Seen store** (`handlers.SeenStore`): persists processed message IDs as
  one JSON file per day under `DATA_DIR`, keeping today plus yesterday in
  memory and pruning files older than two days. Writes are atomic
  (temp file + rename), so duplicates are suppressed across restarts without
  risking corrupted state.

## Entry point

`cmd/bot/main.go` is wiring only: load config → build logger → create client →
validate whitelist entries → build router → register commands and handlers →
start reaction watcher → start pprof sidecar (loopback-only, shut down
gracefully) → block on `client.Start(ctx)` until SIGINT/SIGTERM.

## Dependencies

| Package | Purpose |
| --- | --- |
| `github.com/bwmarrin/discordgo` | Discord API wrapper |
| `github.com/joho/godotenv` | `.env` loading |
| `go.uber.org/goleak` | Goroutine leak detection in tests (dev only) |
| `honnef.co/go/tools/cmd/staticcheck` | Linter pinned via `tool` directive |

Everything else is stdlib (`context`, `log/slog`, `net/http/pprof`,
`encoding/json`, ...).

## Testing

- Unit tests for config parsing, middleware decisions, summary rendering,
  mention handling, and the seen store's rotation/pruning behavior.
- `goleak.VerifyTestMain` guards the handlers package against goroutine leaks.
- CI runs `go vet`, staticcheck, tests with `-race -shuffle=on`, and
  govulncheck.

## Security notes

- The token lives only in `.env` (gitignored) or the process environment;
  never in code, never baked into images.
- An empty whitelist means public mode is an explicit, visible configuration
  choice rather than a forgotten default.
- The pprof endpoint binds to `127.0.0.1` only and must stay internal.
- Message content is truncated before being echoed back into channels.

## Deliberate omissions

- Database persistence for seen reactions: daily JSON files suffice for a
  single instance; add storage when running multi-instance.
- OpenTelemetry/metrics: add when you need dashboards, not before.
- Rate limiting beyond the single-worker watcher: discordgo handles basic
  backoff; add a semaphore if you add bursty workloads.
- Global command registration: guild-scoped keeps iteration fast; revisit for
  multi-guild support.
