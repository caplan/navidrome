# Model Layer - Claude Code Guide

## Overview

This package defines all data models and repository interfaces. Implementations live in `persistence/`.

## Key Types

- `MediaFile` - A song/track (60+ fields including metadata, file info, annotations)
- `Album`, `Artist` - Aggregated from MediaFile data during scanning
- `Playlist` - User playlists with smart playlist support
- `Span` - User-specific markers in songs with position and tags
- `SpanTag` - Predefined tags attachable to spans
- `Tag` - Music metadata tags (genre, mood, etc.) - different from SpanTag
- `Annotations` - User-specific play counts, ratings, stars (embedded in Album/Artist/MediaFile)
- `Bookmarkable` - User-specific position bookmarks

## Conventions

- Repository interfaces are named `<Type>Repository`
- All repositories are accessed via `DataStore` interface
- User-scoped data uses `request.UserFrom(ctx)` in the persistence layer
- Struct tags: `structs:"column_name"` for DB mapping, `json:"fieldName"` for API
- IDs: `id.NewRandom()` for random IDs, `id.NewHash(...)` for deterministic IDs

## Adding a New Model

1. Create model file with struct and repository interface
2. Add accessor method to `DataStore` interface in `datastore.go`
3. Implement in `persistence/` package
4. Add mock to `tests/mock_data_store.go`
