-- +goose Up
CREATE TABLE IF NOT EXISTS span_tag (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL COLLATE NOCASE,
    description TEXT NOT NULL DEFAULT '',
    created_at DATETIME,
    updated_at DATETIME,
    UNIQUE(name)
);

CREATE TABLE IF NOT EXISTS span (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES user(id) ON DELETE CASCADE,
    media_file_id TEXT NOT NULL REFERENCES media_file(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    position REAL NOT NULL,
    end_position REAL,
    created_at DATETIME,
    updated_at DATETIME
);

CREATE INDEX IF NOT EXISTS idx_span_user_media ON span(user_id, media_file_id);

CREATE TABLE IF NOT EXISTS span_span_tag (
    span_id TEXT NOT NULL REFERENCES span(id) ON DELETE CASCADE,
    span_tag_id TEXT NOT NULL REFERENCES span_tag(id) ON DELETE CASCADE,
    PRIMARY KEY (span_id, span_tag_id)
);

-- +goose Down
DROP TABLE IF EXISTS span_span_tag;
DROP TABLE IF EXISTS span;
DROP TABLE IF EXISTS span_tag;
