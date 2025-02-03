package scavenge

import (
	"context"
	"fmt"
	"io"
	"net/http"
)

// HttpClient defines a generic interface for http clients.
type HttpClient interface {
	Do(ctx context.Context, request *Request) (*Response, error)
}

// StandardHttpClient implements [HttpClient] using the standard library's http client.
type StandardHttpClient struct {
	client *http.Client
}

// NewStandardHttpClient creates a [StandardHttpClient]
func NewStandardHttpClient(client *http.Client) StandardHttpClient {
	return StandardHttpClient{client: client}
}

func (c StandardHttpClient) Do(ctx context.Context, request *Request) (*Response, error) {
	req, err := http.NewRequestWithContext(ctx, request.Method, request.Url, request.Body)
	if err != nil {
		return nil, fmt.Errorf("new http request: %w", err)
	}
	res, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do http request: %w", err)
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("read http body: %w", err)
	}
	endUrl := req.URL
	loc, err := res.Location()
	if err == nil {
		endUrl = loc
	}

	return &Response{
		request: request,
		url:     endUrl,
		headers: res.Header,
		body:    body,
	}, nil
}

// FailedRequest is a failed request.
type FailedRequest struct {
	Request *Request
	Error   error
}

// Downloader contains the higher level logic of middleware, scheduling, rate-limiting, and retrying requests.
type Downloader struct {
	client    HttpClient
	responses chan *Response
	fails     chan FailedRequest
}

// DownloaderHandlers runs before a request or a response
type DownloaderHandlers struct {
	HandleRequest  func(request *Request) error
	HandleResponse func(response *Response) error
}

// DownloaderMiddleware creates a [DownloaderHandlers]
type DownloaderMiddleware func(downloader Downloader, next DownloaderHandlers) DownloaderHandlers

type downloaderConfig struct {
	middleware []DownloaderMiddleware
}
type downloaderOption func(cfg *downloaderConfig)

func WithDownloaderMiddleware(middleware ...DownloaderMiddleware) downloaderOption {
	return func(cfg *downloaderConfig) {
		cfg.middleware = middleware
	}
}

// NewDownloader creates a [Downloader].
func NewDownloader(client HttpClient, configurators ...downloaderOption) Downloader {
	downloader := Downloader{
		client:    client,
		responses: make(chan *Response),
		fails:     make(chan FailedRequest),
	}
	return downloader
}

// Queue queues a request for downloading.
func (d Downloader) Queue(req *Request) {
}

// Responses returns the channel of responses as they come in.
func (d Downloader) Responses() chan *Response {
	return d.responses
}
