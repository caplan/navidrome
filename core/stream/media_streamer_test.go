package stream_test

import (
	"context"
	"io"
	"os"

	"github.com/caplan/navidrome/conf"
	"github.com/caplan/navidrome/conf/configtest"
	"github.com/caplan/navidrome/core/stream"
	"github.com/caplan/navidrome/log"
	"github.com/caplan/navidrome/model"
	"github.com/caplan/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("MediaStreamer", func() {
	var streamer stream.MediaStreamer
	var ds model.DataStore
	ffmpeg := tests.NewMockFFmpeg("fake data")
	ctx := log.NewContext(context.TODO())

	BeforeEach(func() {
		DeferCleanup(configtest.SetupConfig())
		conf.Server.CacheFolder, _ = os.MkdirTemp("", "file_caches")
		conf.Server.TranscodingCacheSize = "100MB"
		ds = &tests.MockDataStore{MockedTranscoding: &tests.MockTranscodingRepo{}}
		ds.MediaFile(ctx).(*tests.MockMediaFileRepo).SetData(model.MediaFiles{
			{ID: "123", Path: "tests/fixtures/test.mp3", Suffix: "mp3", BitRate: 128, Duration: 257.0},
		})
		testCache := stream.NewTranscodingCache()
		Eventually(func() bool { return testCache.Available(context.TODO()) }).Should(BeTrue())
		streamer = stream.NewMediaStreamer(ds, ffmpeg, testCache)
	})
	AfterEach(func() {
		_ = os.RemoveAll(conf.Server.CacheFolder)
	})

	Context("NewStream", func() {
		var mf *model.MediaFile
		BeforeEach(func() {
			var err error
			mf, err = ds.MediaFile(ctx).Get("123")
			Expect(err).ToNot(HaveOccurred())
		})
		It("returns a seekable stream if format is 'raw'", func() {
			s, err := streamer.NewStream(ctx, mf, stream.Request{Format: "raw"})
			Expect(err).ToNot(HaveOccurred())
			Expect(s.Seekable()).To(BeTrue())
		})
		It("returns a seekable stream if no format is specified (direct play)", func() {
			s, err := streamer.NewStream(ctx, mf, stream.Request{})
			Expect(err).ToNot(HaveOccurred())
			Expect(s.Seekable()).To(BeTrue())
		})
		It("returns a NON seekable stream if transcode is required", func() {
			s, err := streamer.NewStream(ctx, mf, stream.Request{Format: "mp3", BitRate: 64})
			Expect(err).To(BeNil())
			Expect(s.Seekable()).To(BeFalse())
			Expect(s.Duration()).To(Equal(float32(257.0)))
		})
		It("returns a seekable stream if the file is complete in the cache", func() {
			s, err := streamer.NewStream(ctx, mf, stream.Request{Format: "mp3", BitRate: 32})
			Expect(err).To(BeNil())
			_, _ = io.ReadAll(s)
			_ = s.Close()
			Eventually(func() bool { return ffmpeg.IsClosed() }, "3s").Should(BeTrue())

			s, err = streamer.NewStream(ctx, mf, stream.Request{Format: "mp3", BitRate: 32})
			Expect(err).To(BeNil())
			Expect(s.Seekable()).To(BeTrue())
		})
	})
})
