package downloader

import (
	"context"
	"fmt"
)

// Middleware runs before a request or a response, if either [HandleRequest] or [HandleResponse]
// return an error, the request will be aborted.
type Middleware interface {
	HandleRequest(ctx context.Context, dl Downloader, req *Request) (*Request, error)
	HandleResponse(ctx context.Context, dl Downloader, res *Response) (*Response, error)
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

// NewDownloader creates a [Downloader].
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

// Client returns the [Client] the downloader uses.
func (d Downloader) Client() Client {
	return d.client
}

// Queue queues a request for downloading.
func (d Downloader) Download(ctx context.Context, req *Request) (*Response, error) {
	var err error
	for _, mid := range d.cfg.middleware {
		req, err = mid.HandleRequest(ctx, d, req)
		if err != nil {
			return nil, fmt.Errorf("req middleware: %w", err)
		}
	}

	res, err := d.client.Do(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("http: %w", err)
	}

	for _, mid := range d.cfg.middleware {
		res, err = mid.HandleResponse(ctx, d, res)
		if err != nil {
			return nil, fmt.Errorf("resp middleware: %w", err)
		}
	}

	return res, nil
}
