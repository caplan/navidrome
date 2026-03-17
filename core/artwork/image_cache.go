package artwork

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/caplan/navidrome/conf"
	"github.com/caplan/navidrome/consts"
	"github.com/caplan/navidrome/model"
	"github.com/caplan/navidrome/utils/cache"
	"github.com/caplan/navidrome/utils/singleton"
)

type cacheKey struct {
	artID      model.ArtworkID
	lastUpdate time.Time
}

func (k *cacheKey) Key() string {
	return fmt.Sprintf(
		"%s-%s.%d",
		k.artID.Kind,
		k.artID.ID,
		k.lastUpdate.UnixMilli(),
	)
}

type imageCache struct {
	cache.FileCache
}

func GetImageCache() cache.FileCache {
	return singleton.GetInstance(func() *imageCache {
		return &imageCache{
			FileCache: cache.NewFileCache("Image", conf.Server.ImageCacheSize, consts.ImageCacheDir, consts.DefaultImageCacheMaxItems,
				func(ctx context.Context, arg cache.Item) (io.Reader, error) {
					r, _, err := arg.(artworkReader).Reader(ctx)
					return r, err
				}),
		}
	})
}
