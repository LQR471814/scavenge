package scavenge

import (
	"io"
	"net/http"
	"net/url"
	"strings"
)

// Request represents a standard HTTP request.
type Request struct {
	Method  string
	Url     string
	Headers http.Header
	Body    io.Reader
}

// SetHeader sets an http header on the request.
func (r *Request) SetHeader(key, value string) *Request {
	r.Headers.Set(key, value)
	return r
}

// SetContentType sets the content type of the request body. It sets the `Content-Type` http header.
func (r *Request) SetContentType(mimetype string) *Request {
	if r.Headers == nil {
		r.Headers = http.Header{}
	}
	r.Headers.Set("Content-Type", mimetype)
	return r
}

// SetBody sets the body of the request to an [io.Reader] without changing content-type.
func (r *Request) SetBody(mimetype string, body io.Reader) {
	r.SetContentType(mimetype)
	r.Body = body
}

// SetBodyURLEncodedForm sets the body and content-type of the request to an
// [application/x-www-url-encoded form](https://developer.mozilla.org/en-US/docs/Web/HTTP/Methods/POST)
func (r *Request) SetBodyURLEncodedForm(form url.Values) {
	r.SetContentType("application/x-www-form-urlencoded")
	r.Body = strings.NewReader(form.Encode())
}

// SetBodyMultipartForm sets the body and content-type of the request to an
// [multipart/form-data](https://developer.mozilla.org/en-US/docs/Web/HTTP/Methods/POST)
//
// Note: you'll need to create the [contents] parameter with a [bytes.Buffer] and a [multipart.NewWriter] in a manner like follows:
//
//	var buffer bytes.Buffer
//	writer := multipart.NewWriter(&buffer)
//	defer writer.Close()
//
//	err := writer.WriteField("key" ,"value")
//
//	file, err = os.Open("some_file.txt")
//	defer file.Close()
//
//	fileWriter, err = writer.CreateFormFile("key", "some_file.txt")
//
//	_, err = io.Copy(fileWriter, file)
func (r *Request) SetBodyMultipartForm(contents io.Reader) {
	r.SetContentType("multipart/form-data")
	r.Body = contents
}

// SetBodyJSON sets the body of the request to a json string.
func (r *Request) SetBodyJSON(json string) {
	r.SetContentType("application/json")
	r.Body = strings.NewReader(json)
}

// NewRequest creates a new request with no headers or body.
func NewRequest(method, url string) *Request {
	return &Request{Method: method, Url: url}
}

// GetRequest returns a GET request with no headers or body.
func GetRequest(url string) *Request {
	return &Request{Method: http.MethodGet, Url: url}
}

// PostRequest returns a POST request with no headers or body.
func PostRequest(url string) *Request {
	return &Request{Method: http.MethodPost, Url: url}
}

// DeleteRequest returns a DELETE request with no headers or body.
func DeleteRequest(url string) *Request {
	return &Request{Method: http.MethodDelete, Url: url}
}

// PatchRequest returns a PATCH request with no headers or body.
func PatchRequest(url string) *Request {
	return &Request{Method: http.MethodPatch, Url: url}
}

// PutRequest returns a PUT request with no headers or body.
func PutRequest(url string) *Request {
	return &Request{Method: http.MethodPut, Url: url}
}

// HeadRequest returns a HEAD request with no headers or body.
func HeadRequest(url string) *Request {
	return &Request{Method: http.MethodHead, Url: url}
}
