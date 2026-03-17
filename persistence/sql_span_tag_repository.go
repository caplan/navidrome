package persistence

import (
	"context"
	"strings"
	"time"

	. "github.com/Masterminds/squirrel"
	"github.com/caplan/navidrome/model"
	"github.com/caplan/navidrome/model/id"
	"github.com/pocketbase/dbx"
)

type spanTagRepository struct {
	sqlRepository
}

func NewSpanTagRepository(ctx context.Context, db dbx.Builder) model.SpanTagRepository {
	r := &spanTagRepository{}
	r.ctx = ctx
	r.db = db
	r.tableName = "span_tag"
	return r
}

func (r *spanTagRepository) Add(tag *model.SpanTag) error {
	tag.Name = strings.ToLower(strings.TrimSpace(tag.Name))
	if tag.Name == "" || len(tag.Name) > 64 {
		return model.ErrValidation
	}
	now := time.Now()
	if tag.ID == "" {
		tag.ID = id.NewRandom()
	}
	tag.CreatedAt = now
	tag.UpdatedAt = now

	sq := Insert(r.tableName).SetMap(map[string]any{
		"id":          tag.ID,
		"name":        tag.Name,
		"description": tag.Description,
		"created_at":  tag.CreatedAt,
		"updated_at":  tag.UpdatedAt,
	})
	_, err := r.executeSQL(sq)
	return err
}

func (r *spanTagRepository) GetAll() (model.SpanTags, error) {
	sq := Select("*").From(r.tableName).OrderBy("name")
	var tags model.SpanTags
	err := r.queryAll(sq, &tags)
	return tags, err
}

func (r *spanTagRepository) Get(tagID string) (*model.SpanTag, error) {
	sq := Select("*").From(r.tableName).Where(Eq{"id": tagID})
	var tag model.SpanTag
	err := r.queryOne(sq, &tag)
	if err != nil {
		return nil, err
	}
	return &tag, nil
}

func (r *spanTagRepository) Delete(tagID string) error {
	sq := Delete(r.tableName).Where(Eq{"id": tagID})
	_, err := r.executeSQL(sq)
	return err
}
