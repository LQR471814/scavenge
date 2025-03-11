package downloader

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/url"

	"golang.org/x/net/html"
)

// Response represents a standard HTTP response with some convenience methods.
type Response struct {
	request *Request
	status  int
	url     *url.URL
	headers http.Header
	body    []byte

	directBody io.Reader
}

func NewResponse(request *Request, status int, url *url.URL, headers http.Header, body []byte) *Response {
	return &Response{
		status:  status,
		request: request,
		url:     url,
		headers: headers,
		body:    body,
	}
}

// Request returns the request associated with this response.
func (r *Response) Request() *Request {
	return r.request
}

// Status returns the HTTP status for the response.
func (r *Response) Status() int {
	return r.status
}

// Url returns the url (after redirecting) the response came from.
func (r *Response) Url() *url.URL {
	return r.url
}

// Headers returns the headers of the response.
func (r *Response) Headers() http.Header {
	return r.headers
}

// ContentType returns the mimetype in the `content-type` header.
func (r *Response) ContentType() string {
	if r.headers == nil {
		return ""
	}
	return r.headers.Get("Content-Type")
}

// DirectBody returns the underlying [io.Reader] for direct usage.
//
// Note that this will only be non-nil when the DirectResponse field on the corresponding
// request is set to true.
//
// Middleware should not read from this field, as it is not certain that the reader
// can be read from more than once.
func (r *Response) DirectBody() io.Reader {
	return r.directBody
}

// RawBody returns the raw body contents.
func (r *Response) RawBody() []byte {
	return r.body
}

// JsonBody attempts to interpret the body as json and unmarshal it into the v output pointer.
func (r *Response) JsonBody(v any) error {
	return json.Unmarshal(r.body, v)
}

// HtmlBody attempts to interpret the body as html and parse it into a [html.Node]
func (r *Response) HtmlBody() (*html.Node, error) {
	return html.Parse(bytes.NewBuffer(r.body))
}
