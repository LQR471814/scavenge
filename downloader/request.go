package downloader

import (
	"io"
	"net/http"
	"net/url"
)

// Request represents a standard HTTP request.
type Request struct {
	Method  string
	Url     *url.URL
	Headers http.Header
	Body    []byte

	// DirectBody should be used to pipe an [io.Reader] directly into an HTTP request's body,
	// bypassing all external processing done by middleware or user code.
	//
	// This should only be read from the final step in making the request, as it is not
	// certain that the given [io.Reader] can be read from more than once.
	DirectBody io.Reader

	// DirectResponse can be set to true to indicate that the response body should not be
	// read into a byteslice but remain an [io.Reader] for direct usage.
	DirectResponse bool
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
func (r *Request) SetBody(mimetype string, body []byte) {
	r.SetContentType(mimetype)
	r.Body = body
}

// SetBodyURLEncodedForm sets the body and content-type of the request to an
// [application/x-www-url-encoded form](https://developer.mozilla.org/en-US/docs/Web/HTTP/Methods/POST)
func (r *Request) SetBodyURLEncodedForm(form url.Values) {
	r.SetContentType("application/x-www-form-urlencoded")
	r.Body = []byte(form.Encode())
}

// SetBodyMultipartForm sets the body and content-type of the request to an
// [multipart/form-data](https://developer.mozilla.org/en-US/docs/Web/HTTP/Methods/POST)
//
// Note: you'll need to create the contents parameter with a [bytes.Buffer] and a [multipart.NewWriter] in a manner like follows:
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
//
//	req.SetBodyMultipartForm(buffer.Bytes())
func (r *Request) SetBodyMultipartForm(contents []byte) {
	r.SetContentType("multipart/form-data")
	r.Body = contents
}

// SetBodyJSON sets the body of the request to a json string.
func (r *Request) SetBodyJSON(json string) {
	r.SetContentType("application/json")
	r.Body = []byte(json)
}

// MustParseUrl attempts to the parse the given rawUrl into a [*url.URL]
// if an error is encountered, it panics.
func MustParseUrl(rawUrl string) *url.URL {
	res, err := url.Parse(rawUrl)
	if err != nil {
		panic(err)
	}
	return res
}

// NewRequest creates a new request with no headers or body.
func NewRequest(method string, url *url.URL) *Request {
	return &Request{Method: method, Url: url}
}

// GETRequest returns a GET request with no headers or body.
func GETRequest(url *url.URL) *Request {
	return &Request{Method: http.MethodGet, Url: url}
}

// POSTRequest returns a POST request with no headers or body.
func POSTRequest(url *url.URL) *Request {
	return &Request{Method: http.MethodPost, Url: url}
}

// DELETERequest returns a DELETE request with no headers or body.
func DELETERequest(url *url.URL) *Request {
	return &Request{Method: http.MethodDelete, Url: url}
}

// PATCHRequest returns a PATCH request with no headers or body.
func PATCHRequest(url *url.URL) *Request {
	return &Request{Method: http.MethodPatch, Url: url}
}

// PUTRequest returns a PUT request with no headers or body.
func PUTRequest(url *url.URL) *Request {
	return &Request{Method: http.MethodPut, Url: url}
}

// HEADRequest returns a HEAD request with no headers or body.
func HEADRequest(url *url.URL) *Request {
	return &Request{Method: http.MethodHead, Url: url}
}
