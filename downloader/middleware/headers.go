package middleware

import (
	"context"
	"net/http"

	"github.com/LQR471814/scavenge/downloader"
)

// Headers overrides the given headers with the given header values in each request.
type Headers struct {
	headers http.Header
}

func NewHeaders(headers http.Header) *Headers {
	return &Headers{
		headers: headers,
	}
}

func (h *Headers) HandleRequest(ctx context.Context, req *downloader.Request, meta downloader.RequestMetadata) (*downloader.Response, error) {
	if req.Headers == nil {
		req.Headers = http.Header{}
	}
	for k, values := range h.headers {
		req.Headers.Del(k)
		for _, v := range values {
			req.Headers.Add(k, v)
		}
	}
	return nil, nil
}

func (h *Headers) HandleResponse(ctx context.Context, res *downloader.Response, meta downloader.ResponseMetadata) error {
	return nil
}
