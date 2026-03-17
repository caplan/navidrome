package subsonic

import (
	"net/http"

	"github.com/caplan/navidrome/model"
	"github.com/caplan/navidrome/server/nativeapi"
	"github.com/caplan/navidrome/server/subsonic/responses"
	"github.com/caplan/navidrome/utils/req"
	"github.com/caplan/navidrome/utils/slice"
)

func (api *Router) GetSpanTags(r *http.Request) (*responses.Subsonic, error) {
	ctx := r.Context()
	tags, err := api.ds.SpanTag(ctx).GetAll()
	if err != nil {
		return nil, err
	}

	response := newResponse()
	response.SpanTags = &responses.SpanTags{
		SpanTag: slice.Map(tags, func(t model.SpanTag) responses.SpanTag {
			return responses.SpanTag{
				ID:          t.ID,
				Name:        t.Name,
				Description: t.Description,
			}
		}),
	}
	return response, nil
}

func (api *Router) AddSpanTag(r *http.Request) (*responses.Subsonic, error) {
	p := req.Params(r)
	name, err := p.String("name")
	if err != nil {
		return nil, err
	}
	description, _ := p.String("description")

	tag := &model.SpanTag{
		Name:        name,
		Description: description,
	}
	if err := api.ds.SpanTag(r.Context()).Add(tag); err != nil {
		return nil, err
	}
	return newResponse(), nil
}

func (api *Router) GetSpans(r *http.Request) (*responses.Subsonic, error) {
	p := req.Params(r)
	mediaFileID, err := p.String("id")
	if err != nil {
		return nil, err
	}

	ctx := r.Context()
	spans, err := api.ds.Span(ctx).GetByMediaFile(mediaFileID)
	if err != nil {
		return nil, err
	}

	response := newResponse()
	response.Spans = &responses.Spans{
		Span: slice.Map(spans, spanToResponse),
	}
	return response, nil
}

func (api *Router) AddSpan(r *http.Request) (*responses.Subsonic, error) {
	p := req.Params(r)
	mediaFileID, err := p.String("id")
	if err != nil {
		return nil, err
	}
	name, err := p.String("name")
	if err != nil {
		return nil, err
	}
	position, err := p.Float64("position")
	if err != nil {
		return nil, err
	}

	endPosition, endErr := p.Float64("endPosition")
	var endPtr *float64
	if endErr == nil {
		endPtr = &endPosition
	}

	tagIDs, _ := p.Strings("tagId")
	var tags model.SpanTags
	for _, tagID := range tagIDs {
		tag, err := api.ds.SpanTag(r.Context()).Get(tagID)
		if err != nil {
			return nil, newError(responses.ErrorGeneric, "invalid tag ID: %s", tagID)
		}
		tags = append(tags, *tag)
	}

	span := &model.Span{
		MediaFileID: mediaFileID,
		Name:        name,
		Position:    position,
		EndPosition: endPtr,
		Tags:        tags,
	}

	ctx := r.Context()
	if err := api.ds.Span(ctx).Add(span); err != nil {
		return nil, err
	}

	// Write spans to file metadata (best effort)
	nativeapi.WriteSpansToFile(api.ds, ctx, mediaFileID)

	return newResponse(), nil
}

func (api *Router) DeleteSpan(r *http.Request) (*responses.Subsonic, error) {
	p := req.Params(r)
	spanID, err := p.String("id")
	if err != nil {
		return nil, err
	}

	ctx := r.Context()

	// Get the span to know which media file to update
	span, err := api.ds.Span(ctx).Get(spanID)
	if err != nil {
		return nil, newError(responses.ErrorDataNotFound, "span not found")
	}
	mediaFileID := span.MediaFileID

	if err := api.ds.Span(ctx).Delete(spanID); err != nil {
		return nil, err
	}

	nativeapi.WriteSpansToFile(api.ds, ctx, mediaFileID)

	return newResponse(), nil
}

func spanToResponse(s model.Span) responses.Span {
	return responses.Span{
		ID:          s.ID,
		MediaFileID: s.MediaFileID,
		Name:        s.Name,
		Position:    s.Position,
		EndPosition: s.EndPosition,
		Tags: slice.Map(s.Tags, func(t model.SpanTag) responses.SpanTag {
			return responses.SpanTag{
				ID:          t.ID,
				Name:        t.Name,
				Description: t.Description,
			}
		}),
		Created: s.CreatedAt,
		Changed: s.UpdatedAt,
	}
}

