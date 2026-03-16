# Server - Claude Code Guide

## API Architecture

Navidrome has two API layers:

### Subsonic API (`subsonic/`)
- Standard Subsonic/OpenSubsonic protocol for client compatibility
- Handler signature: `func(r *http.Request) (*responses.Subsonic, error)`
- Parameters via `req.Params(r)` (query string)
- Response types in `responses/responses.go`
- Routes registered in `api.go` using `h(r, "endpointName", api.Handler)`
- XML/JSON/JSONP output based on `f` parameter

### Native API (`nativeapi/`)
- Modern REST API used by the web UI
- JSON request/response
- Generic CRUD via `api.R(r, "/path", model.Type{}, persistable)` for simple entities
- Custom handlers return `http.HandlerFunc` closures
- Routes mounted under `/api/`

## Custom Endpoints

### Spans
- Native: `GET/POST /api/span-tag`, `GET/DELETE /api/span-tag/{id}`
- Native: `POST /api/span`, `GET/PUT/DELETE /api/span/{id}`, `GET /api/song/{id}/spans`
- Subsonic: `getSpanTags`, `addSpanTag`, `getSpans`, `addSpan`, `deleteSpan`

### Visualizations
- Native: `GET /api/song/{id}/visualizations` — JSON with availability per mode
- Native: `GET /api/song/{id}/visualization/{mode}` — SVG download (mode: radio, blocky, ribbon, heatmap)
- Subsonic: `getVisualizationStatus?id=<songId>` — availability with per-mode status

## Adding Endpoints

### Subsonic API
1. Add response type to `responses/responses.go`
2. Add response field to `Subsonic` struct
3. Create handler method on `Router` in a new or existing file
4. Register route in `api.go` with `h(r, "name", api.Handler)`

### Native API
1. Create handler functions as `func(ds model.DataStore) http.HandlerFunc`
2. Add route method on `Router` (e.g., `addMyRoute(r chi.Router)`)
3. Call route method in `routes()` in `native_api.go`

## Authentication
- Subsonic: `authenticate(api.ds)` middleware + `getPlayer(api.players)`
- Native: `server.Authenticator(api.ds)` + `server.JWTRefresher`
- Admin-only: `adminOnlyMiddleware`
- User from context: `request.UserFrom(r.Context())`
