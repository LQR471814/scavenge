package middleware

import (
	"context"
	"encoding/gob"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"scavenge"
	"scavenge/downloader"
	"sync"

	"github.com/zeebo/xxh3"
)

// ReplayStore is an abstract interface various cache storing mechanism can implement to be able to be
// used in a Cache.
type ReplayStore interface {
	Has(ctx context.Context, id string) bool
	Get(ctx context.Context, id string) *downloader.Response
	Set(ctx context.Context, id string, res *downloader.Response)
}

// MemoryReplayStore implements CacheStore with an in-memory [sync.Map]
type MemoryReplayStore struct {
	store sync.Map
}

// NewMemoryReplayStore creates a MemoryCacheStore.
func NewMemoryReplayStore() *MemoryReplayStore {
	return &MemoryReplayStore{
		store: sync.Map{},
	}
}

func (s *MemoryReplayStore) Get(ctx context.Context, id string) *downloader.Response {
	v, ok := s.store.Load(id)
	if !ok {
		return nil
	}
	res := v.(*downloader.Response)
	return res
}

func (s *MemoryReplayStore) Has(ctx context.Context, id string) bool {
	_, ok := s.store.Load(id)
	if !ok {
		return false
	}
	return true
}

func (s *MemoryReplayStore) Set(ctx context.Context, id string, res *downloader.Response) {
	s.store.Store(id, res)
}

type rawResponse struct {
	Status  int
	Request downloader.Request
	Url     *url.URL
	Headers http.Header
	Body    []byte
}

func (r rawResponse) Response() *downloader.Response {
	return downloader.NewResponse(&r.Request, r.Status, r.Url, r.Headers, r.Body)
}

func newRawResponse(r *downloader.Response) rawResponse {
	return rawResponse{
		Status:  r.Status(),
		Request: *r.Request(),
		Url:     r.Url(),
		Headers: r.Headers(),
		Body:    r.RawBody(),
	}
}

// FSReplayStore implements CacheStore with the local filesystem.
type FSReplayStore struct {
	dir string
}

// NewFSReplayStore creates a new FSReplayStore.
func NewFSReplayStore(dir string) FSReplayStore {
	return FSReplayStore{dir: dir}
}

func (s FSReplayStore) filepath(id string) string {
	filename := fmt.Sprint(xxh3.Hash([]byte(id)))
	path := filepath.Join(s.dir, filename)
	return path
}

func (s FSReplayStore) Get(ctx context.Context, id string) *downloader.Response {
	logger := scavenge.GetLoggerFromContext(ctx)

	path := s.filepath(id)
	f, err := os.Open(path)
	if err != nil {
		logger.Error("fs_cache_store", "open file", "path", path, "err", err)
		return nil
	}
	defer f.Close()

	decoder := gob.NewDecoder(f)
	rr := rawResponse{}
	err = decoder.Decode(&rr)
	if err != nil {
		logger.Error("fs_cache_store", "decode response", "path", path, "err", err)
		return nil
	}

	return rr.Response()
}

func (s FSReplayStore) Has(ctx context.Context, id string) bool {
	logger := scavenge.GetLoggerFromContext(ctx)
	path := s.filepath(id)
	_, err := os.Stat(path)
	if err != nil {
		logger.Error("fs_cache_store", "stat file", "path", path, "err", err)
		return false
	}
	return true
}

func (s FSReplayStore) Set(ctx context.Context, id string, res *downloader.Response) {
	logger := scavenge.GetLoggerFromContext(ctx)

	path := s.filepath(id)
	f, err := os.Create(path)
	if err != nil {
		logger.Error("fs_cache_store", "open file for writing", "path", path, "err", err)
		return
	}
	defer f.Close()

	encoder := gob.NewEncoder(f)
	err = encoder.Encode(newRawResponse(res))
	if err != nil {
		logger.Error("fs_cache_store", "encode response", "path", path, "err", err)
	}
}
