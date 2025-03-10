package downloader

import (
	"context"
	"fmt"
	"net/url"
	"time"
)

// RequestMetadata represents additional information that may be useful to middleware.
type RequestMetadata struct {
	Referer   *url.URL
	AttemptNo int
}

// ResponseMetadata represents additional information that may be useful to middleware.
type ResponseMetadata struct {
	RequestMetadata
	Elapsed time.Duration
}

// Middleware runs before a request or a response, if either HandleRequest or HandleResponse
// return an error, the request will be aborted.
//
// if HandleRequest returns a non-nil Response, it will be used as the response for the request
// and the rest of the request middlewares will be skipped, this response will also bypass response
// middlewares.
type Middleware interface {
	HandleRequest(ctx context.Context, req *Request, meta RequestMetadata) (*Response, error)
	HandleResponse(ctx context.Context, res *Response, meta ResponseMetadata) error
}

// DroppedRequest indicates that the given request has been dropped and should not be retried.
func DroppedRequest(reason error) error {
	return fmt.Errorf("dropped request: %w", reason)
}
