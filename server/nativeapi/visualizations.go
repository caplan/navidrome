package nativeapi

import (
	"encoding/json"
	"net/http"
	"os"
	"slices"

	"github.com/go-chi/chi/v5"
	"github.com/navidrome/navidrome/core/visualization"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/server"
)

func (api *Router) addVisualizationRoute(r chi.Router) {
	r.With(server.URLParamsMiddleware).Get("/song/{id}/visualizations", getVisualizationStatus(api.ds))
	r.With(server.URLParamsMiddleware).Get("/song/{id}/visualization/{mode}", getVisualization(api.ds))
}

type vizStatus struct {
	Available      bool            `json:"available"`
	AcousticID     string          `json:"acousticId,omitempty"`
	SpecVersion    string          `json:"specVersion"`
	Visualizations map[string]bool `json:"visualizations"`
}

func getVisualizationStatus(ds model.DataStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		songID := chi.URLParam(r, "id")

		mf, err := ds.MediaFile(ctx).Get(songID)
		if err != nil {
			http.Error(w, "song not found", http.StatusNotFound)
			return
		}

		status := vizStatus{
			SpecVersion:    visualization.SpecVersion,
			Visualizations: make(map[string]bool, len(visualization.Modes)),
		}

		if mf.AcousticID == "" {
			for _, mode := range visualization.Modes {
				status.Visualizations[mode] = false
			}
		} else {
			status.AcousticID = mf.AcousticID
			anyAvailable := false
			for _, mode := range visualization.Modes {
				path := visualization.GetVisualizationPath(mf.AcousticID, mode)
				_, err := os.Stat(path)
				exists := err == nil
				status.Visualizations[mode] = exists
				if exists {
					anyAvailable = true
				}
			}
			status.Available = anyAvailable
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(status)
	}
}

func getVisualization(ds model.DataStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		songID := chi.URLParam(r, "id")
		mode := chi.URLParam(r, "mode")

		if !slices.Contains(visualization.Modes, mode) {
			http.Error(w, "invalid visualization mode", http.StatusBadRequest)
			return
		}

		mf, err := ds.MediaFile(ctx).Get(songID)
		if err != nil {
			http.Error(w, "song not found", http.StatusNotFound)
			return
		}
		if mf.AcousticID == "" {
			http.Error(w, "acoustic ID not yet calculated for this song", http.StatusNotFound)
			return
		}

		svgPath := visualization.GetVisualizationPath(mf.AcousticID, mode)
		data, err := os.ReadFile(svgPath)
		if err != nil {
			if os.IsNotExist(err) {
				http.Error(w, "visualization not yet generated", http.StatusNotFound)
				return
			}
			log.Error(ctx, "Error reading visualization", "path", svgPath, err)
			http.Error(w, "error reading visualization", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "image/svg+xml")
		w.Header().Set("Cache-Control", "public, max-age=86400")
		_, _ = w.Write(data)
	}
}
