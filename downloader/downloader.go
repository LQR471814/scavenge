package downloader

import (
	"context"
	"fmt"
	"time"
)

type ResponseMetadata struct {
	Elapsed time.Duration
}

// Middleware runs before a request or a response, if either [HandleRequest] or [HandleResponse]
// return an error, the request will be aborted.
//
// if HandleRequest returns a non-nil Response, it will be used as the response for the request
// and the rest of the request middlewares will be skipped, this response will also bypass response
// middlewares.
type Middleware interface {
	HandleRequest(ctx context.Context, dl Downloader, req *Request) (*Response, error)
	HandleResponse(ctx context.Context, dl Downloader, res *Response, meta ResponseMetadata) error
}

type config struct {
	middleware []Middleware
}
type option func(cfg *config)

// WithMiddleware appends the given middleware to the list of registered middlewares.
func WithMiddleware(middleware ...Middleware) option {
	return func(cfg *config) {
		cfg.middleware = append(cfg.middleware, middleware...)
	}
}

type Downloader struct {
	client Client
	cfg    config
}

// NewDownloader creates a Downloader.
func NewDownloader(client Client, options ...option) Downloader {
	cfg := config{}
	for _, option := range options {
		option(&cfg)
	}

	downloader := Downloader{
		client: client,
		cfg:    cfg,
	}
	return downloader
}

// Client returns the Client the downloader uses.
func (d Downloader) Client() Client {
	return d.client
}

// Queue queues a request for downloading.
func (d Downloader) Download(ctx context.Context, req *Request) (*Response, error) {
	for _, mid := range d.cfg.middleware {
		res, err := mid.HandleRequest(ctx, d, req)
		if err != nil {
			return nil, fmt.Errorf("req middleware: %w", err)
		}
		if res != nil {
			return res, nil
		}
	}

	t1 := time.Now()
	res, err := d.client.Do(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("http: %w", err)
	}
	t2 := time.Now()

	meta := ResponseMetadata{
		Elapsed: t2.Sub(t1),
	}

	for _, mid := range d.cfg.middleware {
		err = mid.HandleResponse(ctx, d, res, meta)
		if err != nil {
			return nil, fmt.Errorf("resp middleware: %w", err)
		}
	}

	return res, nil
}
