# Design Document: Discord Bot Template (Go)

## Project Overview

A modular Discord bot template built with `bwmarrin/discordgo` that serves as a foundation for future Discord integration projects. The design emphasizes separation of concerns, clean middleware patterns, and minimal coupling between Discord-specific logic and business logic.

**Module**: `github.com/teyhouse/discord-bot-golang`  
**Go Version**: 1.22+ (per-iteration loop variables, range-over-int)

---

## Architecture

```
discord-bot-golang/
├── cmd/bot/main.go           # Entry point, wiring only
├── internal/
│   ├── config/               # Configuration loading (.env)
│   ├── discord/              # Discord-specific logic (core)
│   │   ├── client.go         # Bot session management
│   │   ├── commands/         # Slash command handlers
│   │   │   ├── ping.go
│   │   │   └── whisper.go
│   │   ├── middleware/       # Permission middleware
│   │   │   └── whitelist.go
│   │   ├── handlers/         # Event handlers
│   │   │   ├── reactions.go  # :eyes: reaction watcher
│   │   │   └── mentions.go   # @bot mention handler
│   │   └── router.go         # Command/event routing
│   └── logger/               # Structured logging (slog)
├── .env.example              # Template env file
├── go.mod
├── go.sum
└── Makefile
```

### Package Organization Principles

- **Domain over layers**: Each package owns its types + logic (`commands/`, `middleware/`, `handlers/`)
- **No `utils/` or `common/`**: Packages named for what they *provide*
- **Interfaces with consumer**: Small (1-3 methods), discovered when second implementation appears
- **Privacy via `internal/`**: Compiler-enforced boundaries

---

## Configuration (`.env`)

```env
# Required
BOT_TOKEN=your_bot_token_here
CHANNEL_ID=123456789012345678
GUILD_ID=123456789012345678

# Optional (empty = public bot, no whitelist)
WHITELISTED_USER_IDS=138372729660243968,1521565722487033866
```

### Config Package (`internal/config`)

```go
type Config struct {
    BotToken         string
    ChannelID        string
    GuildID          string
    WhitelistedUsers []string // empty = disabled (public)
}
```

- Load via `github.com/joho/godotenv` (stdlib-compatible)
- Validate required fields at startup, fail fast
- Parse `WHITELISTED_USER_IDS` as comma-separated, trim spaces
- Empty whitelist → middleware becomes no-op (public bot)

---

## Core Discord Client (`internal/discord/client.go`)

### Responsibilities

- Session lifecycle: connect, disconnect, reconnect handling
- Intent configuration: `GuildMessages`, `GuildMessageReactions`, `DirectMessages`, `MessageContent`
- Graceful shutdown on `SIGINT`/`SIGTERM` with context cancellation
- Expose `*discordgo.Session` for handlers (read-only access)

### Implementation

```go
type Client struct {
    session *discordgo.Session
    config  *config.Config
    logger  *slog.Logger
}

func NewClient(cfg *config.Config, logger *slog.Logger) (*Client, error)
func (c *Client) Start(ctx context.Context) error
func (c *Client) Stop() error
func (c *Client) Session() *discordgo.Session // for handlers
```

- Use `context.WithCancel` for shutdown coordination
- Register handlers via `session.AddHandler()`
- Enable pprof sidecar on `127.0.0.1:6060` (internal only)

---

## Permission Middleware (`internal/discord/middleware/whitelist.go`)

### Design Goal

Optional, composable middleware that wraps command handlers. Easy to remove for public commands.

### Interface

```go
type PermissionChecker interface {
    Allowed(userID string) bool
}

type WhitelistChecker struct {
    allowed map[string]struct{} // O(1) lookup
}

func NewWhitelistChecker(userIDs []string) *WhitelistChecker
func (w *WhitelistChecker) Allowed(userID string) bool
```

### Middleware Wrapper

```go
// Wraps a handler, returns a new handler that checks permission first
func WithPermissionCheck(checker PermissionChecker, next HandlerFunc) HandlerFunc {
    return func(s *discordgo.Session, i *discordgo.InteractionCreate) {
        if !checker.Allowed(i.Member.User.ID) {
            respondEphemeral(s, i, "Not authorized")
            return
        }
        next(s, i)
    }
}
```

- `HandlerFunc` type: `func(s *discordgo.Session, i *discordgo.InteractionCreate)`
- Empty whitelist → `Allowed()` always returns `true` (public mode)
- Ephemeral response for denied access (private to user)

---

## Slash Commands (`internal/discord/commands/`)

### Command Registration

Register commands on startup for the configured `GuildID` (immediate availability). Use `ApplicationCommandCreate`.

```go
var Commands = []*discordgo.ApplicationCommand{
    {
        Name:        "ping",
        Description: "Responds with Pong",
    },
    {
        Name:        "whisper",
        Description: "Sends a private message only you can see",
    },
}
```

### Ping Command (`ping.go`)

```go
func PingHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
    respond(s, i, "Pong", false) // public response
}
```

- Public response (visible to all in channel)

### Whisper Command (`whisper.go`)

```go
func WhisperHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
    respond(s, i, "pssst, only visible for you", true) // ephemeral
}
```

- Ephemeral response (only visible to command author)

### Response Helper

```go
func respond(s *discordgo.Session, i *discordgo.InteractionCreate, content string, ephemeral bool) {
    flags := discordgo.MessageFlagsEphemeral
    if !ephemeral {
        flags = 0
    }
    s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
        Type: discordgo.InteractionResponseChannelMessageWithSource,
        Data: &discordgo.InteractionResponseData{
            Content: content,
            Flags:   flags,
        },
    })
}
```

---

## Event Handlers (`internal/discord/handlers/`)

### Reaction Watcher (`reactions.go`)

**Requirement**: Periodically scan channel for messages in last 5 minutes with :eyes: reaction, quote and respond "I saw that!"

#### Design

- Background goroutine with ticker (configurable interval, default 30s)
- Bounded concurrency: single worker, uses context for cancellation
- Fetch messages via `ChannelMessages` (limit 100, before latest)
- Filter: timestamp > `time.Now().Add(-5*time.Minute)`
- Check reactions for `👀` (Unicode U+1F440) or custom `:eyes:` emoji
- Avoid duplicate responses: track processed message IDs in memory (with TTL cleanup)

```go
type ReactionWatcher struct {
    session    *discordgo.Session
    channelID  string
    interval   time.Duration
    seen       *sync.Map // messageID -> timestamp
    logger     *slog.Logger
}

func NewReactionWatcher(s *discordgo.Session, channelID string, interval time.Duration, logger *slog.Logger) *ReactionWatcher
func (w *ReactionWatcher) Start(ctx context.Context)
func (w *ReactionWatcher) checkOnce(ctx context.Context)
```

- `sync.Map` for concurrent-safe seen-tracking
- Periodic cleanup of entries older than 10 minutes
- Quote format: `> {author}: {content}\n\nI saw that!`

### Mention Handler (`mentions.go`)

**Requirement**: When bot is @mentioned directly, respond with summary of original message.

```go
func MentionHandler(s *discordgo.Session, m *discordgo.MessageCreate) {
    if m.Author.ID == s.State.User.ID {
        return // ignore self
    }
    if !mentionsBot(s, m) {
        return
    }
    summary := summarize(m.Content)
    s.ChannelMessageSendReply(m.ChannelID, summary, m.Reference())
}
```

- `mentionsBot()`: checks `m.Mentions` for bot's user ID
- `summarize()`: truncate to 200 chars, strip markdown, add "Summary: " prefix
- Use `MessageSendReply` for threaded context

---

## Router (`internal/discord/router.go`)

Central registration point for all handlers and commands.

```go
type Router struct {
    client     *Client
    config     *config.Config
    logger     *slog.Logger
    checker    PermissionChecker
}

func NewRouter(client *Client, cfg *config.Config, logger *slog.Logger) *Router
func (r *Router) RegisterCommands() error
func (r *Router) RegisterHandlers()
```

- Wraps command handlers with `WithPermissionCheck` by default
- Public commands can opt-out: `router.RegisterPublicCommand("cmd", handler)`
- Registers event handlers: `MessageCreate`, `MessageReactionAdd`

---

## Main Entry Point (`cmd/bot/main.go`)

```go
func main() {
    // 1. Load config
    cfg, err := config.Load()
    if err != nil { slog.Error("config load failed", "err", err); os.Exit(1) }

    // 2. Logger
    logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
    slog.SetDefault(logger)

    // 3. Discord client
    client, err := discord.NewClient(cfg, logger)
    if err != nil { logger.Error("client init failed", "err", err); os.Exit(1) }

    // 4. Permission checker
    var checker PermissionChecker = &discord.WhitelistChecker{}
    if len(cfg.WhitelistedUsers) > 0 {
        checker = discord.NewWhitelistChecker(cfg.WhitelistedUsers)
    }

    // 5. Router
    router := discord.NewRouter(client, cfg, logger, checker)
    router.RegisterCommands()
    router.RegisterHandlers()

    // 6. Start
    ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
    defer cancel()

    // Reaction watcher
    watcher := discord.NewReactionWatcher(client.Session(), cfg.ChannelID, 30*time.Second, logger)
    go watcher.Start(ctx)

    // Pprof sidecar
    go func() {
        logger.Error("pprof exited", "err", http.ListenAndServe("127.0.0.1:6060", nil))
    }()

    if err := client.Start(ctx); err != nil {
        logger.Error("bot stopped", "err", err)
    }
}
```

---

## Dependencies

| Package | Purpose |
|---------|---------|
| `github.com/bwmarrin/discordgo` | Discord API wrapper |
| `github.com/joho/godotenv` | .env loading |
| `go.uber.org/goleak` | Test goroutine leak detection (dev) |
| `honnef.co/go/tools/cmd/staticcheck` | Linting (tool directive) |

**Stdlib only**: `context`, `sync`, `time`, `log/slog`, `net/http/pprof`, `os`, `os/signal`, `syscall`, `strings`, `fmt`

---

## CI / Tooling (Makefile)

```make
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
	go build -o bin/bot ./cmd/bot
```

- `go.mod` includes `toolchain go1.22` for reproducible builds
- `go get -tool honnef.co/go/tools/cmd/staticcheck` for pinned linter

---

## Testing Strategy

- **Unit tests**: Middleware logic, config parsing, summarization (no Discord dependency)
- **Integration tests**: Require real token — skip in CI, document manual run
- **Leak detection**: `goleak.VerifyTestMain` in `internal/discord/handlers/reactions_test.go`
- **Fuzzing**: `FuzzSummarize` for mention handler input parsing

---

## .env.example (Task)

Generate at project root:

```env
# Your bot token from https://discord.com/developers/applications
BOT_TOKEN=

# The channel where the bot posts
# Right-click channel → Copy ID (enable Developer Mode in Discord settings)
CHANNEL_ID=

# Comma separated Discord user IDs (optional)
# Right-click your name → Copy ID
# Leave empty for public bot (no permission checks)
WHITELISTED_USER_IDS=

# Your Discord server (guild) ID — makes commands appear immediately after restart
# Right-click the server name in the channel list → Copy ID
GUILD_ID=
```

---

## Implementation Order

1. **Project init**: `go mod init github.com/teyhouse/discord-bot-golang`, add dependencies
2. **Config package**: `.env` loading, validation, `WHITELISTED_USER_IDS` parsing
3. **Logger package**: slog JSON setup
4. **Discord client**: Session management, intents, graceful shutdown
5. **Permission middleware**: WhitelistChecker, WithPermissionCheck wrapper
6. **Commands**: Ping, Whisper + registration
7. **Handlers**: Reaction watcher (ticker + context), Mention handler
8. **Router**: Central registration, command wrapping
9. **Main**: Wiring, signal handling, pprof sidecar
10. **CI/Makefile**: Lint, test, vuln, run targets
11. **.env.example**: Template file
12. **Tests**: Unit tests for config, middleware, summarization; goleak for watcher

---

## Future Extensibility Points

- **Additional commands**: Add to `commands/`, register in router
- **New event handlers**: Add to `handlers/`, register in router
- **Different permission models**: Implement `PermissionChecker` interface (role-based, DB-backed, etc.)
- **Persistence**: Add `internal/storage/` package for seen messages, config
- **Metrics**: Add OpenTelemetry + Prometheus exporter (debugging-and-observability.md)
- **Rate limiting**: Semaphore in client for outbound API calls

---

## Security Considerations

- Token only in `.env` (gitignored), never in code
- `WHITELISTED_USER_IDS` empty = public bot (explicit opt-in to restriction)
- No `http.DefaultClient` — Discord library handles HTTP; ensure library uses timeouts
- Input validation: slash command inputs are typed by Discord; message content truncated before processing
- Pprof only on `127.0.0.1:6060`

---

## Open Questions

1. **Reaction emoji**: Use Unicode `👀` only, or support custom `:eyes:` emoji by name? (Unicode simpler, no guild emoji fetch needed) A: Unicode for now
2. **Duplicate detection**: In-memory `sync.Map` with TTL is simple but loses state on restart. Acceptable for template? A: Track sate in a local .json file for the current day, likely with the messageId? Goal: only store the current + last day, removed everything older than 2 days 
3. **Command sync**: Guild-only (instant) vs global (1hr propagate). Guild-only chosen per requirements. A: Yes, guild-only is fine as long as it is fast propagated on startup
4. **Log level**: Default JSON to stdout. Add `LOG_LEVEL` env var for debug/prod? A: Add Log-Level yes


---

## Ponytail Notes

- **Skipped**: Database persistence for seen reactions (in-memory with TTL). Add when: multi-instance deployment needed.
- **Skipped**: OpenTelemetry wiring. Add when: distributed tracing required.
- **Skipped**: Role-based permissions. Add when: whitelist insufficient.
- **Skipped**: Rate limiter on outbound Discord calls. Add when: hitting API limits.
- **Skipped**: Global command registration fallback. Add when: multi-guild support needed.
