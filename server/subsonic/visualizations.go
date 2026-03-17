package subsonic

import (
	"net/http"
	"os"

	"github.com/caplan/navidrome/core/visualization"
	"github.com/caplan/navidrome/server/subsonic/responses"
	"github.com/caplan/navidrome/utils/req"
)

func (api *Router) GetVisualizationStatus(r *http.Request) (*responses.Subsonic, error) {
	p := req.Params(r)
	songID, err := p.String("id")
	if err != nil {
		return nil, err
	}

	ctx := r.Context()
	mf, err := api.ds.MediaFile(ctx).Get(songID)
	if err != nil {
		return nil, newError(responses.ErrorDataNotFound, "song not found")
	}

	status := responses.VisualizationStatus{
		Available:   false,
		SpecVersion: visualization.SpecVersion,
	}

	if mf.AcousticID != "" {
		status.AcousticID = mf.AcousticID
		for _, mode := range visualization.Modes {
			path := visualization.GetVisualizationPath(mf.AcousticID, mode)
			_, statErr := os.Stat(path)
			exists := statErr == nil
			status.Types = append(status.Types, responses.VisualizationType{
				Mode:      mode,
				Available: exists,
			})
			if exists {
				status.Available = true
			}
		}
	} else {
		for _, mode := range visualization.Modes {
			status.Types = append(status.Types, responses.VisualizationType{
				Mode:      mode,
				Available: false,
			})
		}
	}

	response := newResponse()
	response.VisualizationStatus = &status
	return response, nil
}
