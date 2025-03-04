package middleware

import (
	"context"
	"encoding/gob"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sync"

	"github.com/LQR471814/scavenge"
	"github.com/LQR471814/scavenge/downloader"

	"github.com/zeebo/xxh3"
)

// ReplayStore is an abstract interface various cache storing mechanism can implement to be able to be
// used in a Cache.
type ReplayStore interface {
	Has(ctx context.Context, session, id string) bool
	// Get should return nil if a stored request with the given id does not yet exist.
	Get(ctx context.Context, session, id string) *downloader.Response
	Set(ctx context.Context, session, id string, res *downloader.Response)
}

// MemoryReplayStore implements CacheStore with an in-memory [sync.Map]
type MemoryReplayStore struct {
	store sync.Map
}

func NewMemoryReplayStore() *MemoryReplayStore {
	return &MemoryReplayStore{
		store: sync.Map{},
	}
}

func (s *MemoryReplayStore) Get(ctx context.Context, session, id string) *downloader.Response {
	v, ok := s.store.Load(session + id)
	if !ok {
		return nil
	}
	res := v.(*downloader.Response)
	return res
}

func (s *MemoryReplayStore) Has(ctx context.Context, session, id string) bool {
	_, ok := s.store.Load(session + id)
	if !ok {
		return false
	}
	return true
}

func (s *MemoryReplayStore) Set(ctx context.Context, session, id string, res *downloader.Response) {
	s.store.Store(session+id, res)
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

func NewFSReplayStore(dir string) FSReplayStore {
	return FSReplayStore{dir: dir}
}

func (s FSReplayStore) filepath(session, id string) (dir string) {
	filename := fmt.Sprint(xxh3.Hash([]byte(id)))
	path := filepath.Join(s.dir, session, filename)
	return path
}

func (s FSReplayStore) Get(ctx context.Context, session, id string) *downloader.Response {
	logger := scavenge.LoggerFromContext(ctx)

	path := s.filepath(session, id)
	f, err := os.Open(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		logger.Warn("fs_cache_store", "open file", "path", path, "err", err)
		return nil
	}
	defer f.Close()

	decoder := gob.NewDecoder(f)
	rr := rawResponse{}
	err = decoder.Decode(&rr)
	if err != nil {
		logger.Warn("fs_cache_store", "decode response", "path", path, "err", err)
		return nil
	}

	return rr.Response()
}

func (s FSReplayStore) Has(ctx context.Context, session, id string) bool {
	logger := scavenge.LoggerFromContext(ctx)
	path := s.filepath(session, id)
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false
	}
	if err != nil {
		logger.Warn("fs_cache_store", "stat file", "path", path, "err", err)
		return false
	}
	return true
}

func (s FSReplayStore) Set(ctx context.Context, session, id string, res *downloader.Response) {
	logger := scavenge.LoggerFromContext(ctx)

	dir := filepath.Join(s.dir, session)
	err := os.MkdirAll(dir, 0777)
	if err != nil {
		logger.Warn("fs_cache_store", "make session dir", "dir", dir, "err", err)
		return
	}

	path := s.filepath(session, id)
	f, err := os.Create(path)
	if err != nil {
		logger.Warn("fs_cache_store", "open file for writing", "path", path, "err", err)
		return
	}
	defer f.Close()

	encoder := gob.NewEncoder(f)
	err = encoder.Encode(newRawResponse(res))
	if err != nil {
		logger.Warn("fs_cache_store", "encode response", "path", path, "err", err)
	}
}
