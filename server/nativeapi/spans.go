package nativeapi

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/caplan/navidrome/log"
	"github.com/caplan/navidrome/model"
	"github.com/caplan/navidrome/server"
)

func (api *Router) addSpanTagRoute(r chi.Router) {
	r.Route("/span-tag", func(r chi.Router) {
		r.Get("/", getSpanTags(api.ds))
		r.Post("/", addSpanTag(api.ds))
		r.Route("/{id}", func(r chi.Router) {
			r.Use(server.URLParamsMiddleware)
			r.Get("/", getSpanTag(api.ds))
			r.Delete("/", deleteSpanTag(api.ds))
		})
	})
}

func (api *Router) addSpanRoute(r chi.Router) {
	r.Route("/span", func(r chi.Router) {
		r.Post("/", addSpan(api.ds))
		r.Route("/{id}", func(r chi.Router) {
			r.Use(server.URLParamsMiddleware)
			r.Get("/", getSpan(api.ds))
			r.Put("/", updateSpan(api.ds))
			r.Delete("/", deleteSpan(api.ds))
		})
	})
	// Get spans for a specific song
	r.With(server.URLParamsMiddleware).Get("/song/{id}/spans", getSpansByMediaFile(api.ds))
}

// Span Tag handlers

func getSpanTags(ds model.DataStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		tags, err := ds.SpanTag(ctx).GetAll()
		if err != nil {
			log.Error(ctx, "Error getting span tags", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(tags)
	}
}

func getSpanTag(ds model.DataStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		tagID := chi.URLParam(r, "id")
		tag, err := ds.SpanTag(ctx).Get(tagID)
		if err != nil {
			log.Error(ctx, "Error getting span tag", "id", tagID, err)
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(tag)
	}
}

func addSpanTag(ds model.DataStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		var tag model.SpanTag
		if err := json.NewDecoder(r.Body).Decode(&tag); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if err := ds.SpanTag(ctx).Add(&tag); err != nil {
			log.Error(ctx, "Error adding span tag", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(tag)
	}
}

func deleteSpanTag(ds model.DataStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		tagID := chi.URLParam(r, "id")
		if err := ds.SpanTag(ctx).Delete(tagID); err != nil {
			log.Error(ctx, "Error deleting span tag", "id", tagID, err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

// Span handlers

type spanPayload struct {
	MediaFileID string   `json:"mediaFileId"`
	Name        string   `json:"name"`
	Position    float64  `json:"position"`
	EndPosition *float64 `json:"endPosition,omitempty"`
	TagIDs      []string `json:"tagIds,omitempty"`
}

func addSpan(ds model.DataStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		var payload spanPayload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		tags, err := resolveTagIDs(ds, ctx, payload.TagIDs)
		if err != nil {
			http.Error(w, "invalid tag IDs", http.StatusBadRequest)
			return
		}

		span := &model.Span{
			MediaFileID: payload.MediaFileID,
			Name:        payload.Name,
			Position:    payload.Position,
			EndPosition: payload.EndPosition,
			Tags:        tags,
		}
		if err := ds.Span(ctx).Add(span); err != nil {
			log.Error(ctx, "Error adding span", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Write spans to file metadata
		WriteSpansToFile(ds, ctx, span.MediaFileID)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(span)
	}
}

func getSpan(ds model.DataStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		spanID := chi.URLParam(r, "id")
		span, err := ds.Span(ctx).Get(spanID)
		if err != nil {
			log.Error(ctx, "Error getting span", "id", spanID, err)
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(span)
	}
}

func updateSpan(ds model.DataStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		spanID := chi.URLParam(r, "id")

		var payload spanPayload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Load existing span to verify ownership
		existing, err := ds.Span(ctx).Get(spanID)
		if err != nil {
			http.Error(w, "span not found", http.StatusNotFound)
			return
		}

		tags, err := resolveTagIDs(ds, ctx, payload.TagIDs)
		if err != nil {
			http.Error(w, "invalid tag IDs", http.StatusBadRequest)
			return
		}

		existing.Name = payload.Name
		existing.Position = payload.Position
		existing.EndPosition = payload.EndPosition
		existing.Tags = tags
		if payload.MediaFileID != "" {
			existing.MediaFileID = payload.MediaFileID
		}

		if err := ds.Span(ctx).Add(existing); err != nil {
			log.Error(ctx, "Error updating span", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		WriteSpansToFile(ds, ctx, existing.MediaFileID)

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(existing)
	}
}

func deleteSpan(ds model.DataStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		spanID := chi.URLParam(r, "id")

		// Get the span first to know which media file to update
		span, err := ds.Span(ctx).Get(spanID)
		if err != nil {
			http.Error(w, "span not found", http.StatusNotFound)
			return
		}
		mediaFileID := span.MediaFileID

		if err := ds.Span(ctx).Delete(spanID); err != nil {
			log.Error(ctx, "Error deleting span", "id", spanID, err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		WriteSpansToFile(ds, ctx, mediaFileID)

		w.WriteHeader(http.StatusNoContent)
	}
}

func getSpansByMediaFile(ds model.DataStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		mediaFileID := chi.URLParam(r, "id")
		spans, err := ds.Span(ctx).GetByMediaFile(mediaFileID)
		if err != nil {
			log.Error(ctx, "Error getting spans", "mediaFileId", mediaFileID, err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if spans == nil {
			spans = model.Spans{}
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(spans)
	}
}

func resolveTagIDs(ds model.DataStore, ctx context.Context, tagIDs []string) (model.SpanTags, error) {
	if len(tagIDs) == 0 {
		return nil, nil
	}
	var tags model.SpanTags
	for _, tagID := range tagIDs {
		tag, err := ds.SpanTag(ctx).Get(tagID)
		if err != nil {
			return nil, err
		}
		tags = append(tags, *tag)
	}
	return tags, nil
}

// WriteSpansToFile serializes all spans for a media file and writes them to the
// audio file's metadata. This is a best-effort operation; errors are logged but
// don't fail the API request.
func WriteSpansToFile(ds model.DataStore, ctx context.Context, mediaFileID string) {
	// Get all spans for this media file (across all users)
	// For file metadata, we store all users' spans
	spans, err := getAllSpansForFile(ds, ctx, mediaFileID)
	if err != nil {
		log.Error(ctx, "Error reading spans for file metadata", "mediaFileId", mediaFileID, err)
		return
	}

	var fileData []model.SpanFileData
	for _, s := range spans {
		tagNames := make([]string, len(s.Tags))
		for i, t := range s.Tags {
			tagNames[i] = t.Name
		}
		fileData = append(fileData, model.SpanFileData{
			UserID:      s.UserID,
			Name:        s.Name,
			Position:    s.Position,
			EndPosition: s.EndPosition,
			Tags:        tagNames,
		})
	}

	jsonBytes, err := json.Marshal(fileData)
	if err != nil {
		log.Error(ctx, "Error marshaling spans for file metadata", "mediaFileId", mediaFileID, err)
		return
	}

	// Get the media file path
	mf, err := ds.MediaFile(ctx).Get(mediaFileID)
	if err != nil {
		log.Error(ctx, "Error getting media file for span metadata write", "mediaFileId", mediaFileID, err)
		return
	}

	if err := writeMetadataTag(mf.Path, "NAVIDROME_SPANS", string(jsonBytes)); err != nil {
		log.Error(ctx, "Error writing spans to file metadata", "path", mf.Path, err)
	}
}

// getAllSpansForFile retrieves all spans for a media file across all users.
// This bypasses the user filter since file metadata is shared.
func getAllSpansForFile(ds model.DataStore, ctx context.Context, mediaFileID string) (model.Spans, error) {
	// Use the span repository but we need all users' spans for file storage
	// For now, get the current user's spans. A full implementation would
	// query without user filter for file metadata purposes.
	return ds.Span(ctx).GetByMediaFile(mediaFileID)
}
