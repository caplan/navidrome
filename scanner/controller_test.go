package scanner_test

import (
	"context"

	"github.com/caplan/navidrome/conf/configtest"
	"github.com/caplan/navidrome/consts"
	"github.com/caplan/navidrome/core/artwork"
	"github.com/caplan/navidrome/core/metrics"
	"github.com/caplan/navidrome/core/playlists"
	"github.com/caplan/navidrome/db"
	"github.com/caplan/navidrome/model"
	"github.com/caplan/navidrome/persistence"
	"github.com/caplan/navidrome/scanner"
	"github.com/caplan/navidrome/server/events"
	"github.com/caplan/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Controller", func() {
	var ctx context.Context
	var ds *tests.MockDataStore
	var ctrl model.Scanner

	Describe("Status", func() {
		BeforeEach(func() {
			ctx = context.Background()
			db.Init(ctx)
			DeferCleanup(func() { Expect(tests.ClearDB()).To(Succeed()) })
			DeferCleanup(configtest.SetupConfig())
			ds = &tests.MockDataStore{RealDS: persistence.New(db.Db())}
			ds.MockedProperty = &tests.MockedPropertyRepo{}
			ctrl = scanner.New(ctx, ds, artwork.NoopCacheWarmer(), events.NoopBroker(), playlists.NewPlaylists(ds), metrics.NewNoopInstance())
		})

		It("includes last scan error", func() {
			Expect(ds.Property(ctx).Put(consts.LastScanErrorKey, "boom")).To(Succeed())
			status, err := ctrl.Status(ctx)
			Expect(err).ToNot(HaveOccurred())
			Expect(status.LastError).To(Equal("boom"))
		})

		It("includes scan type and error in status", func() {
			// Set up test data in property repo
			Expect(ds.Property(ctx).Put(consts.LastScanErrorKey, "test error")).To(Succeed())
			Expect(ds.Property(ctx).Put(consts.LastScanTypeKey, "full")).To(Succeed())

			// Get status and verify basic info
			status, err := ctrl.Status(ctx)
			Expect(err).ToNot(HaveOccurred())
			Expect(status.LastError).To(Equal("test error"))
			Expect(status.ScanType).To(Equal("full"))
		})
	})
})
