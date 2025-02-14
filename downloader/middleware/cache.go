package middleware

import (
	"context"
	"net/http"
	"scavenge/downloader"
	"time"

	"github.com/PuerkitoBio/purell"
)

// Cache provides response caching functionality for GET requests (this behavior can be configured
// with the WithShouldCache option).
type Cache struct {
	store CacheStore
	cfg   cacheConfig
}

type cacheHandler = func(req *downloader.Request) (key string, cache bool)

type cacheConfig struct {
	retentionTime time.Duration
	shouldCache   cacheHandler
}

type cacheOption func(cfg *cacheConfig)

// WithCacheRetention sets the cache retention duration for responses (default is 1 hour)
func WithCacheRetention(dur time.Duration) cacheOption {
	return func(cfg *cacheConfig) {
		cfg.retentionTime = dur
	}
}

// WithShouldCache sets the handler for determining whether a request should be cached,
// and computing what the cache key of the request is.
//
// (default checks if the request method is GET and uses the normalized url as the cache key)
func WithShouldCache(shouldCache cacheHandler) cacheOption {
	return func(cfg *cacheConfig) {
		cfg.shouldCache = shouldCache
	}
}

// NewCache creates a new Cache.
func NewCache(store CacheStore, options ...cacheOption) Cache {
	cfg := cacheConfig{
		retentionTime: time.Hour,
		shouldCache: func(req *downloader.Request) (key string, cache bool) {
			if req.Method != http.MethodGet {
				return "", false
			}
			return purell.NormalizeURL(req.Url, purell.FlagsSafe), true
		},
	}
	for _, opt := range options {
		opt(&cfg)
	}
	return Cache{store: store, cfg: cfg}
}

func (c Cache) HandleRequest(ctx context.Context, dl downloader.Downloader, req *downloader.Request) (*downloader.Response, error) {
	hash, shouldCache := c.cfg.shouldCache(req)
	if !shouldCache {
		return nil, nil
	}
	res, lastUpdated := c.store.Get(hash)
	if time.Since(lastUpdated).Seconds() > c.cfg.retentionTime.Seconds() {
		c.store.Evict(hash)
		return nil, nil
	}
	return res, nil
}

func (c Cache) HandleResponse(ctx context.Context, dl downloader.Downloader, res *downloader.Response) error {
	return nil
}
