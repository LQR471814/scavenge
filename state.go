package scavenge

import (
	"io"
	"net/url"
	"os"
	"scavenge/downloader"
	"scavenge/item"
)

type reqJob struct {
	Req     *downloader.Request
	Referer *url.URL
	// attempt is unexported, since all requests should reset their timers after resuming.
	attempt int
}

type state struct {
	reqs  []reqJob
	items []item.Item
}

type StateStore interface {
	Load() (io.ReadCloser, error)
	Store() (io.WriteCloser, error)
	Delete() error
}

type FileStateStore struct {
	path string
}

func NewFileStateStore(path string) FileStateStore {
	return FileStateStore{
		path: path,
	}
}

func (s FileStateStore) Load() (io.ReadCloser, error) {
	return os.OpenFile(s.path, os.O_RDONLY, 0400)
}

func (s FileStateStore) Store() (io.WriteCloser, error) {
	return os.OpenFile(s.path, os.O_WRONLY, 0200)
}
