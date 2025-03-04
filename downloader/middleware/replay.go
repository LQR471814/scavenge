package middleware

import (
	"context"
	"fmt"
	"net/http"

	"github.com/LQR471814/scavenge"
	"github.com/LQR471814/scavenge/downloader"

	"github.com/PuerkitoBio/purell"
)

// Replay provides response replay (and caching) functionality.
type Replay struct {
	sessionId string
	store     ReplayStore
	handler   ReplayHandler
}

// ReplayHandler determines what requests to replay and what the unique key for
// the request should be.
type ReplayHandler = func(ctx context.Context, req *downloader.Request, meta downloader.RequestMetadata) (key string, replay bool)

// ReplayGetRequests is a ReplayHandler that replays all GET requests and uses
// their normalized url as the request key.
func ReplayGetRequests(ctx context.Context, req *downloader.Request, meta downloader.RequestMetadata) (key string, cache bool) {
	if req.Method != http.MethodGet {
		return "", false
	}
	return purell.NormalizeURL(req.Url, purell.FlagsSafe), true
}

func NewReplay(sessionId string, store ReplayStore, handler ReplayHandler) Replay {
	return Replay{
		sessionId: sessionId,
		store:     store,
		handler:   handler,
	}
}

func (c Replay) storeId(key string) string {
	return fmt.Sprintf("%s:%s", c.sessionId, key)
}

func (c Replay) HandleRequest(ctx context.Context, req *downloader.Request, meta downloader.RequestMetadata) (*downloader.Response, error) {
	logger := scavenge.LoggerFromContext(ctx)
	key, replay := c.handler(ctx, req, meta)
	if !replay {
		return nil, nil
	}
	id := c.storeId(key)
	res := c.store.Get(ctx, c.sessionId, id)
	if res == nil {
		return nil, nil
	}
	logger.Debug("replay", "replaying response", "key", key)
	return res, nil
}

func (c Replay) HandleResponse(
	ctx context.Context,
	res *downloader.Response,
	meta downloader.ResponseMetadata,
) error {
	key, replay := c.handler(ctx, res.Request(), meta.RequestMetadata)
	if !replay {
		return nil
	}
	c.store.Set(ctx, c.sessionId, c.storeId(key), res)
	return nil
}
