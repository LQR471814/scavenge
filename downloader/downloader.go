package downloader

import (
	"context"
	"fmt"
	"time"
)

// Downloader is effectively an HTTP client wrapped in middleware.
//
// Note:
//   - Middlewares are evaluated from start to finish for requests and responses.
type Downloader struct {
	client     Client
	middleware []Middleware
}

func NewDownloader(client Client, middleware ...Middleware) Downloader {
	downloader := Downloader{
		client:     client,
		middleware: middleware,
	}
	return downloader
}

// Client returns the Client the downloader uses.
func (d Downloader) Client() Client {
	return d.client
}

// Queue queues a request for downloading.
func (d Downloader) Download(ctx context.Context, req *Request, meta RequestMetadata) (*Response, error) {
	for _, mid := range d.middleware {
		res, err := mid.HandleRequest(ctx, req, meta)
		if err != nil {
			return nil, err
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

	resMeta := ResponseMetadata{
		RequestMetadata: meta,
		Elapsed:         t2.Sub(t1),
	}

	for _, mid := range d.middleware {
		err = mid.HandleResponse(ctx, res, resMeta)
		if err != nil {
			return nil, err
		}
	}

	return res, nil
}
