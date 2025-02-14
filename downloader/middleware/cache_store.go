package middleware

import (
	"fmt"
	"os"
	"path/filepath"
	"scavenge/downloader"
	"scavenge/telemetry"
	"sync"
	"time"

	"github.com/zeebo/xxh3"
)

// CacheStore is an abstract interface various cache storing mechanism can implement to be able to be
// used in a Cache.
type CacheStore interface {
	Get(id string) (res *downloader.Response, lastUpdated time.Time)
	Set(id string, res *downloader.Response)
	Evict(id string)
}

// MemoryCacheStore implements CacheStore with an in-memory [sync.Map]
type MemoryCacheStore struct {
	store sync.Map
}

// NewMemoryCacheStore creates a MemoryCacheStore.
func NewMemoryCacheStore() *MemoryCacheStore {
	return &MemoryCacheStore{
		store: sync.Map{},
	}
}

type memoryCacheEntry struct {
	res         *downloader.Response
	lastUpdated time.Time
}

func (s *MemoryCacheStore) Get(id string) (res *downloader.Response, lastUpdated time.Time) {
	v, ok := s.store.Load(id)
	if !ok {
		return nil, time.Time{}
	}
	entry := v.(memoryCacheEntry)
	return entry.res, entry.lastUpdated
}

func (s *MemoryCacheStore) Set(id string, res *downloader.Response) {
	s.store.Store(id, res)
}

func (s *MemoryCacheStore) Evict(id string) {
	s.store.Delete(id)
}

// FSCacheStore implements CacheStore with the local filesystem.
type FSCacheStore struct {
	dir string
}

func NewFSCacheStore(dir string) FSCacheStore {
	return FSCacheStore{dir: dir}
}

func (s FSCacheStore) Get(id string) (res *downloader.Response, lastUpdated time.Time) {
	filename := fmt.Sprint(xxh3.Hash([]byte(id)))
	path := filepath.Join(s.dir, filename)

	f, err := os.Open(path)
	if err != nil {
		telemetry.Error("fs_cache_store", "open file", "path", path, "err", err)
		return nil, time.Time{}
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		telemetry.Error("fs_cache_store", "stat file", "path", path, "err", err)
		return nil, time.Time{}
	}

	return &downloader.Response{}, info.ModTime()
}
