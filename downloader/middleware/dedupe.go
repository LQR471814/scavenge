package middleware

import (
	"context"
	"fmt"
	"net/http"
	"scavenge/downloader"
	"sync"

	"github.com/PuerkitoBio/purell"
)

// Dedupe drops duplicate GET requests, requests are differentiated by their normalized url.
type Dedupe struct {
	reqs sync.Map
}

func NewDedupe() *Dedupe {
	return &Dedupe{
		reqs: sync.Map{},
	}
}

func (d *Dedupe) HandleRequest(ctx context.Context, req *downloader.Request, meta downloader.RequestMetadata) (*downloader.Response, error) {
	if req.Method != http.MethodGet {
		return nil, nil
	}
	// only deduplicate requests on their first try
	if meta.AttemptNo > 0 {
		return nil, nil
	}
	normalized := purell.NormalizeURL(req.Url, purell.FlagsSafe)
	_, loaded := d.reqs.LoadOrStore(normalized, struct{}{})
	if loaded {
		return nil, downloader.DroppedRequest(fmt.Errorf("duplicate request: GET %s", normalized))
	}
	return nil, nil
}

func (d *Dedupe) HandleResponse(ctx context.Context, res *downloader.Response, meta downloader.ResponseMetadata) error {
	return nil
}
