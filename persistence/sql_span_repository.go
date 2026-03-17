package persistence

import (
	"context"
	"time"

	. "github.com/Masterminds/squirrel"
	"github.com/caplan/navidrome/log"
	"github.com/caplan/navidrome/model"
	"github.com/caplan/navidrome/model/id"
	"github.com/caplan/navidrome/model/request"
	"github.com/pocketbase/dbx"
)

type spanRepository struct {
	sqlRepository
}

func NewSpanRepository(ctx context.Context, db dbx.Builder) model.SpanRepository {
	r := &spanRepository{}
	r.ctx = ctx
	r.db = db
	r.tableName = "span"
	return r
}

func (r *spanRepository) Add(span *model.Span) error {
	user, _ := request.UserFrom(r.ctx)
	span.UserID = user.ID

	if span.Name == "" || len(span.Name) > 64 {
		return model.ErrValidation
	}

	now := time.Now()
	isNew := span.ID == ""
	if isNew {
		span.ID = id.NewRandom()
		span.CreatedAt = now
	}
	span.UpdatedAt = now

	err := r.putSpan(span, isNew)
	if err != nil {
		return err
	}

	// Replace tag associations
	return r.replaceTagAssociations(span.ID, span.Tags)
}

func (r *spanRepository) putSpan(span *model.Span, isNew bool) error {
	values := map[string]any{
		"id":            span.ID,
		"user_id":       span.UserID,
		"media_file_id": span.MediaFileID,
		"name":          span.Name,
		"position":      span.Position,
		"end_position":  span.EndPosition,
		"created_at":    span.CreatedAt,
		"updated_at":    span.UpdatedAt,
	}

	if isNew {
		sq := Insert(r.tableName).SetMap(values)
		_, err := r.executeSQL(sq)
		return err
	}
	sq := Update(r.tableName).SetMap(values).Where(And{
		Eq{"id": span.ID},
		Eq{"user_id": span.UserID},
	})
	_, err := r.executeSQL(sq)
	return err
}

func (r *spanRepository) replaceTagAssociations(spanID string, tags model.SpanTags) error {
	// Delete existing associations
	del := Delete("span_span_tag").Where(Eq{"span_id": spanID})
	if _, err := r.executeSQL(del); err != nil {
		return err
	}

	// Insert new associations
	for _, tag := range tags {
		ins := Insert("span_span_tag").SetMap(map[string]any{
			"span_id":     spanID,
			"span_tag_id": tag.ID,
		})
		if _, err := r.executeSQL(ins); err != nil {
			return err
		}
	}
	return nil
}

func (r *spanRepository) Get(spanID string) (*model.Span, error) {
	user, _ := request.UserFrom(r.ctx)
	sq := Select("*").From(r.tableName).Where(And{
		Eq{"id": spanID},
		Eq{"user_id": user.ID},
	})

	var span model.Span
	err := r.queryOne(sq, &span)
	if err != nil {
		return nil, err
	}

	tags, err := r.loadTags(spanID)
	if err != nil {
		log.Error(r.ctx, "Error loading span tags", "spanId", spanID, err)
	}
	span.Tags = tags
	return &span, nil
}

func (r *spanRepository) GetByMediaFile(mediaFileID string) (model.Spans, error) {
	user, _ := request.UserFrom(r.ctx)
	sq := Select("*").From(r.tableName).Where(And{
		Eq{"media_file_id": mediaFileID},
		Eq{"user_id": user.ID},
	}).OrderBy("position")

	var spans model.Spans
	err := r.queryAll(sq, &spans)
	if err != nil {
		return nil, err
	}

	// Load tags for each span
	for i := range spans {
		tags, err := r.loadTags(spans[i].ID)
		if err != nil {
			log.Error(r.ctx, "Error loading span tags", "spanId", spans[i].ID, err)
			continue
		}
		spans[i].Tags = tags
	}
	return spans, nil
}

func (r *spanRepository) Delete(spanID string) error {
	user, _ := request.UserFrom(r.ctx)
	sq := Delete(r.tableName).Where(And{
		Eq{"id": spanID},
		Eq{"user_id": user.ID},
	})
	_, err := r.executeSQL(sq)
	return err
}

func (r *spanRepository) loadTags(spanID string) (model.SpanTags, error) {
	sq := Select("span_tag.*").From("span_tag").
		Join("span_span_tag ON span_span_tag.span_tag_id = span_tag.id").
		Where(Eq{"span_span_tag.span_id": spanID}).
		OrderBy("span_tag.name")
	var tags model.SpanTags
	err := r.queryAll(sq, &tags)
	return tags, err
}
