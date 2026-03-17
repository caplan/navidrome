package core

import (
	"github.com/google/wire"
	"github.com/caplan/navidrome/core/agents"
	"github.com/caplan/navidrome/core/external"
	"github.com/caplan/navidrome/core/ffmpeg"
	"github.com/caplan/navidrome/core/lyrics"
	"github.com/caplan/navidrome/core/metrics"
	"github.com/caplan/navidrome/core/playback"
	"github.com/caplan/navidrome/core/playlists"
	"github.com/caplan/navidrome/core/scrobbler"
	"github.com/caplan/navidrome/core/stream"
)

var Set = wire.NewSet(
	stream.NewMediaStreamer,
	stream.GetTranscodingCache,
	NewArchiver,
	NewPlayers,
	NewShare,
	playlists.NewPlaylists,
	NewLibrary,
	NewUser,
	NewMaintenance,
	stream.NewTranscodeDecider,
	agents.GetAgents,
	external.NewProvider,
	wire.Bind(new(external.Agents), new(*agents.Agents)),
	ffmpeg.New,
	scrobbler.GetPlayTracker,
	playback.GetInstance,
	metrics.GetInstance,
	lyrics.NewLyrics,
)
