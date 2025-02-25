package scavenge

import (
	"io"
	"net/url"
	"os"
	"path/filepath"
	"scavenge/downloader"
	"scavenge/item"
)

type reqJob struct {
	Req     *downloader.Request
	Referer *url.URL
	// attempt is unexported, since all requests should reset their timers after resuming.
	attempt int
}

type itemJob struct {
	Item    item.Item
	attempt int
}

type state struct {
	Reqs  []reqJob
	Items []itemJob
}

// StateStore is an interface for reading/writing scraping state to some persistent storage.
//
// Scraping state itself is just a byteslice of an arbitrary length.
type StateStore interface {
	Load() (io.ReadCloser, error)
	Store() (io.WriteCloser, error)
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
	f, err := os.OpenFile(s.path, os.O_RDONLY, 0400)
	if os.IsNotExist(err) {
		return nil, nil
	}
	return f, err
}

func (s FileStateStore) Store() (io.WriteCloser, error) {
	os.MkdirAll(filepath.Dir(s.path), 0777)
	return os.Create(s.path)
}
