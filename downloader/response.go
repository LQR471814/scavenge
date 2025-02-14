package downloader

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/url"

	"github.com/PuerkitoBio/goquery"
)

// Response represents a standard HTTP response with some convenience methods.
type Response struct {
	request *Request
	url     *url.URL
	headers http.Header
	body    []byte
}

// Request returns the request associated with this response.
func (r *Response) Request() *Request {
	return r.request
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
	return r.headers.Get("Content-Type")
}

// RawBody returns the raw body contents.
func (r *Response) RawBody() []byte {
	return r.body
}

// JsonBody attempts to interpret the body as json and unmarshal it into the [v] output pointer.
func (r *Response) JsonBody(v any) error {
	return json.Unmarshal(r.body, v)
}

// HtmlBody attempts to interpret the body as html and parse it into a [*goquery.Document].
func (r *Response) HtmlBody() (*goquery.Document, error) {
	return goquery.NewDocumentFromReader(bytes.NewBuffer(r.body))
}
