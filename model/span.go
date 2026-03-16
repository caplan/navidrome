package model

import "time"

// SpanTag is a predefined tag that can be attached to spans.
// Tags must be created before they can be referenced by spans.
type SpanTag struct {
	ID          string    `structs:"id" json:"id"`
	Name        string    `structs:"name" json:"name"`
	Description string    `structs:"description" json:"description"`
	CreatedAt   time.Time `structs:"created_at" json:"createdAt"`
	UpdatedAt   time.Time `structs:"updated_at" json:"updatedAt"`
}

type SpanTags []SpanTag

type SpanTagRepository interface {
	Add(tag *SpanTag) error
	GetAll() (SpanTags, error)
	Get(id string) (*SpanTag, error)
	Delete(id string) error
}

// Span is a user-specific marker in a song at a particular position,
// optionally with an end position. Each span has a name and a list of tags.
type Span struct {
	ID           string    `structs:"id" json:"id"`
	UserID       string    `structs:"user_id" json:"userId"`
	MediaFileID  string    `structs:"media_file_id" json:"mediaFileId"`
	Name         string    `structs:"name" json:"name"`
	Position     float64   `structs:"position" json:"position"`
	EndPosition  *float64  `structs:"end_position" json:"endPosition,omitempty"`
	Tags         SpanTags  `structs:"-" json:"tags,omitempty"`
	CreatedAt    time.Time `structs:"created_at" json:"createdAt"`
	UpdatedAt    time.Time `structs:"updated_at" json:"updatedAt"`
}

type Spans []Span

// SpanFileData is the JSON structure serialized into audio file metadata.
type SpanFileData struct {
	UserID      string   `json:"userId"`
	Name        string   `json:"name"`
	Position    float64  `json:"position"`
	EndPosition *float64 `json:"endPosition,omitempty"`
	Tags        []string `json:"tags,omitempty"` // tag names
}

type SpanRepository interface {
	// Add creates or updates a span for the current user.
	Add(span *Span) error
	// Get returns a span by ID (must belong to current user).
	Get(id string) (*Span, error)
	// GetByMediaFile returns all spans for a media file belonging to the current user.
	GetByMediaFile(mediaFileID string) (Spans, error)
	// Delete removes a span by ID (must belong to current user).
	Delete(id string) error
}
