# Navidrome - Claude Code Guide

## Project Overview

Navidrome is an open-source, self-hosted music streaming server written in Go with a React/TypeScript frontend. It implements the Subsonic API for compatibility with existing music clients and also provides a native REST API.

## Build & Development

```bash
make setup          # Install all dependencies (Go + Node)
make dev            # Start dev server with hot reload (frontend + backend)
make server         # Start backend only with hot reload
make build          # Build complete binary
make buildjs        # Build frontend only
make wire           # Regenerate dependency injection code (required after changing providers)
make gen            # Run all code generation (Wire + plugin PDK)
```

**Build tags**: `netgo,sqlite_fts5` (always required)

```bash
make test                        # Run Go tests
make test PKG=./server/subsonic  # Run tests for specific package
make watch                       # Run tests in watch mode (Ginkgo)
make testall                     # Run all tests (Go + JS + i18n)
make lint                        # Lint with golangci-lint
make format                      # Format code (goimports + prettier)
```

**Important**: After modifying the `DataStore` interface or adding new providers, run `make wire` to regenerate `cmd/wire_gen.go`.

## Architecture

### Directory Structure

| Directory | Purpose |
|-----------|---------|
| `cmd/` | CLI entry points and Wire dependency injection |
| `model/` | Data models and repository interfaces |
| `persistence/` | SQLite repository implementations (squirrel query builder) |
| `db/migrations/` | Goose database migrations (SQL and Go) |
| `server/subsonic/` | Subsonic API handlers and response types |
| `server/nativeapi/` | Native REST API handlers |
| `server/public/` | Public endpoints (shares) |
| `core/` | Business logic services (artwork, streaming, playlists, etc.) |
| `scanner/` | Multi-phase library scanning pipeline |
| `adapters/` | External service integrations (LastFM, Spotify, taglib) |
| `plugins/` | WebAssembly plugin system (Extism) |
| `conf/` | Configuration (Viper/TOML, env vars with `ND_` prefix) |
| `ui/` | React/TypeScript frontend (Vite, Material UI, React-Admin) |
| `tests/` | Shared test utilities and mock implementations |
| `utils/` | Utility packages |

### Key Patterns

- **Repository pattern**: Interfaces in `model/`, implementations in `persistence/`
- **Dependency injection**: Google Wire (compile-time), providers in `core/wire_providers.go`
- **Database**: SQLite with `Masterminds/squirrel` query builder, Goose migrations
- **Testing**: Ginkgo/Gomega (BDD style), mocks in `tests/mock_data_store.go`
- **HTTP router**: go-chi/chi v5
- **Logging**: Context-aware via `log.Info(ctx, "msg", "key", val)`
- **Error handling**: Sentinel errors in `model/errors.go` (`ErrNotFound`, `ErrNotAuthorized`, etc.)
- **User context**: `request.UserFrom(ctx)` to get the logged-in user in repositories

### Adding a New Feature (typical workflow)

1. Define model structs and repository interface in `model/`
2. Create database migration in `db/migrations/` (SQL preferred for simple schemas)
3. Implement repository in `persistence/`
4. Add repository accessor to `DataStore` interface in `model/datastore.go`
5. Wire it up in `persistence/persistence.go`
6. Add mock to `tests/mock_data_store.go`
7. Add API handlers in `server/nativeapi/` and/or `server/subsonic/`
8. Add response types if needed in `server/subsonic/responses/`
9. Run `make wire` if providers changed

### Commit Conventions

```
<type>(<scope>): <description>
```

Types: `feat`, `fix`, `sec`, `docs`, `style`, `refactor`, `perf`, `test`, `build`, `revert`, `chore`

### Important Notes

- The existing `model.Tag` / `model.TagRepository` is for **music metadata tags** (genre, mood, etc.), not user-facing tags
- User-specific data (annotations, bookmarks, spans) uses the pattern of scoping queries by `user_id` from context
- The `taglib` adapter may show `invalid flag in pkg-config` warnings on macOS - this is expected and non-fatal
- Always add DCO sign-off to commits: `git commit --signoff`
