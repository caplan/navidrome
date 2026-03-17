//go:build wireinject

package cmd

import (
	"context"

	"github.com/google/wire"
	"github.com/caplan/navidrome/adapters/lastfm"
	"github.com/caplan/navidrome/adapters/listenbrainz"
	"github.com/caplan/navidrome/core"
	"github.com/caplan/navidrome/core/agents"
	"github.com/caplan/navidrome/core/artwork"
	"github.com/caplan/navidrome/core/lyrics"
	"github.com/caplan/navidrome/core/metrics"
	"github.com/caplan/navidrome/core/playback"
	"github.com/caplan/navidrome/core/scrobbler"
	"github.com/caplan/navidrome/db"
	"github.com/caplan/navidrome/model"
	"github.com/caplan/navidrome/persistence"
	"github.com/caplan/navidrome/plugins"
	"github.com/caplan/navidrome/scanner"
	"github.com/caplan/navidrome/server"
	"github.com/caplan/navidrome/server/events"
	"github.com/caplan/navidrome/server/nativeapi"
	"github.com/caplan/navidrome/server/public"
	"github.com/caplan/navidrome/server/subsonic"
)

var allProviders = wire.NewSet(
	core.Set,
	artwork.Set,
	server.New,
	subsonic.New,
	nativeapi.New,
	public.New,
	persistence.New,
	lastfm.NewRouter,
	listenbrainz.NewRouter,
	events.GetBroker,
	scanner.New,
	scanner.GetWatcher,
	metrics.GetPrometheusInstance,
	db.Db,
	plugins.GetManager,
	wire.Bind(new(agents.PluginLoader), new(*plugins.Manager)),
	wire.Bind(new(scrobbler.PluginLoader), new(*plugins.Manager)),
	wire.Bind(new(lyrics.PluginLoader), new(*plugins.Manager)),
	wire.Bind(new(nativeapi.PluginManager), new(*plugins.Manager)),
	wire.Bind(new(core.PluginUnloader), new(*plugins.Manager)),
	wire.Bind(new(plugins.PluginMetricsRecorder), new(metrics.Metrics)),
	wire.Bind(new(core.Watcher), new(scanner.Watcher)),
)

func CreateDataStore() model.DataStore {
	panic(wire.Build(
		allProviders,
	))
}

func CreateServer() *server.Server {
	panic(wire.Build(
		allProviders,
	))
}

func CreateNativeAPIRouter(ctx context.Context) *nativeapi.Router {
	panic(wire.Build(
		allProviders,
	))
}

func CreateSubsonicAPIRouter(ctx context.Context) *subsonic.Router {
	panic(wire.Build(
		allProviders,
	))
}

func CreatePublicRouter() *public.Router {
	panic(wire.Build(
		allProviders,
	))
}

func CreateLastFMRouter() *lastfm.Router {
	panic(wire.Build(
		allProviders,
	))
}

func CreateListenBrainzRouter() *listenbrainz.Router {
	panic(wire.Build(
		allProviders,
	))
}

func CreateInsights() metrics.Insights {
	panic(wire.Build(
		allProviders,
	))
}

func CreatePrometheus() metrics.Metrics {
	panic(wire.Build(
		allProviders,
	))
}

func CreateScanner(ctx context.Context) model.Scanner {
	panic(wire.Build(
		allProviders,
	))
}

func CreateScanWatcher(ctx context.Context) scanner.Watcher {
	panic(wire.Build(
		allProviders,
	))
}

func GetPlaybackServer() playback.PlaybackServer {
	panic(wire.Build(
		allProviders,
	))
}

func getPluginManager() *plugins.Manager {
	panic(wire.Build(
		allProviders,
	))
}

func GetPluginManager(ctx context.Context) *plugins.Manager {
	manager := getPluginManager()
	manager.SetSubsonicRouter(CreateSubsonicAPIRouter(ctx))
	return manager
}
