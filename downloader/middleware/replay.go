package middleware

import (
	"context"
	"fmt"
	"net/http"
	"scavenge/downloader"

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
type ReplayHandler = func(ctx context.Context, req *downloader.Request) (key string, replay bool)

// ReplayGetRequests is a ReplayHandler that replays all GET requests and uses
// their normalized url as the request key.
func ReplayGetRequests(req *downloader.Request) (key string, cache bool) {
	if req.Method != http.MethodGet {
		return "", false
	}
	return purell.NormalizeURL(req.Url, purell.FlagsSafe), true
}

// NewReplay creates a new Replay.
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

func (c Replay) HandleRequest(ctx context.Context, dl downloader.Downloader, req *downloader.Request) (*downloader.Response, error) {
	key, replay := c.handler(ctx, req)
	if !replay {
		return nil, nil
	}
	res := c.store.Get(ctx, c.storeId(key))
	return res, nil
}

func (c Replay) HandleResponse(
	ctx context.Context,
	dl downloader.Downloader,
	res *downloader.Response,
	meta downloader.ResponseMetadata,
) error {
	key, replay := c.handler(ctx, res.Request())
	if !replay {
		return nil
	}
	c.store.Set(ctx, c.storeId(key), res)
	return nil
}
