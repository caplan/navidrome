package public

import (
	"net/http"
	"path"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/caplan/navidrome/conf"
	"github.com/caplan/navidrome/consts"
	"github.com/caplan/navidrome/core"
	"github.com/caplan/navidrome/core/artwork"
	"github.com/caplan/navidrome/core/publicurl"
	"github.com/caplan/navidrome/core/stream"
	"github.com/caplan/navidrome/log"
	"github.com/caplan/navidrome/model"
	"github.com/caplan/navidrome/server"
	"github.com/caplan/navidrome/ui"
)

type Router struct {
	http.Handler
	artwork       artwork.Artwork
	streamer      stream.MediaStreamer
	archiver      core.Archiver
	share         core.Share
	assetsHandler http.Handler
	ds            model.DataStore
}

func New(ds model.DataStore, artwork artwork.Artwork, streamer stream.MediaStreamer, share core.Share, archiver core.Archiver) *Router {
	p := &Router{ds: ds, artwork: artwork, streamer: streamer, share: share, archiver: archiver}
	shareRoot := path.Join(conf.Server.BasePath, consts.URLPathPublic)
	p.assetsHandler = http.StripPrefix(shareRoot, http.FileServer(http.FS(ui.BuildAssets())))
	p.Handler = p.routes()

	return p
}

func (pub *Router) routes() http.Handler {
	r := chi.NewRouter()

	r.Group(func(r chi.Router) {
		r.Use(server.URLParamsMiddleware)
		r.Group(func(r chi.Router) {
			if conf.Server.DevArtworkMaxRequests > 0 {
				log.Debug("Throttling public images endpoint", "maxRequests", conf.Server.DevArtworkMaxRequests,
					"backlogLimit", conf.Server.DevArtworkThrottleBacklogLimit, "backlogTimeout",
					conf.Server.DevArtworkThrottleBacklogTimeout)
				r.Use(middleware.ThrottleBacklog(conf.Server.DevArtworkMaxRequests, conf.Server.DevArtworkThrottleBacklogLimit,
					conf.Server.DevArtworkThrottleBacklogTimeout))
			}
			r.HandleFunc("/img/{id}", pub.handleImages)
		})
		if conf.Server.EnableSharing {
			r.HandleFunc("/s/{id}", pub.handleStream)
			if conf.Server.EnableDownloads {
				r.HandleFunc("/d/{id}", pub.handleDownloads)
			}
			r.HandleFunc("/{id}/m3u", pub.handleM3U)
			r.HandleFunc("/{id}", pub.handleShares)
			r.HandleFunc("/", pub.handleShares)
			r.Handle("/*", pub.assetsHandler)
		}
	})
	return r
}

func ShareURL(r *http.Request, id string) string {
	uri := path.Join(consts.URLPathPublic, id)
	return publicurl.PublicURL(r, uri, nil)
}
