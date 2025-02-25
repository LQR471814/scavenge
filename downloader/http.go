package downloader

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
)

// Client defines a generic interface for http clients.
type Client interface {
	Do(ctx context.Context, request *Request) (*Response, error)
}

// HttpClient implements Client using the standard library's http client.
type HttpClient struct {
	client *http.Client
}

// NewHttpClient creates a HttpClient
func NewHttpClient(client *http.Client) HttpClient {
	return HttpClient{client: client}
}

// Do implements Client.Do
func (c HttpClient) Do(ctx context.Context, request *Request) (*Response, error) {
	body := request.DirectBody
	if body == nil {
		body = bytes.NewBuffer(request.Body)
	}

	req, err := http.NewRequestWithContext(ctx, request.Method, request.Url.String(), body)
	if err != nil {
		return nil, fmt.Errorf("new http request: %w", err)
	}
	res, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do http request: %w", err)
	}

	var resbody []byte
	var directBody io.Reader
	if !request.DirectResponse {
		resbody, err = io.ReadAll(res.Body)
		if err != nil {
			return nil, fmt.Errorf("read http body: %w", err)
		}
	} else {
		directBody = res.Body
	}

	endUrl := req.URL
	loc, err := res.Location()
	if err == nil {
		endUrl = loc
	}

	return &Response{
		request:    request,
		url:        endUrl,
		headers:    res.Header,
		body:       resbody,
		directBody: directBody,
	}, nil
}
