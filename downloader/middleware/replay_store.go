package middleware

import (
	"bytes"
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

// MetaEncoder takes in an any value used as an element in request metadata and serializes it.
type MetaEncoder interface {
	// Marshal turns an given metadata element into a byteslice.
	//
	// Marshal can return a zero-length byteslice to indicate that the given metadata element should not be serialized.
	Marshal(meta any) ([]byte, error)

	// Unmarshal returns a given metadata element from a byteslice.
	Unmarshal(buff []byte) (any, error)
}

// GobMetaEncoder implements MetaEncoder using [encoding/gob].
type GobMetaEncoder struct{}

// NewGobMetaEncoder creates a MetaEncoder that uses [encoding/gob], all
// the types you expect to be in downloader.Request.Meta should be passed
// as parameters to this function to be registered with [encoding/gob].
func NewGobMetaEncoder(types ...any) GobMetaEncoder {
	for _, t := range types {
		gob.Register(t)

		// test if all types can be serialized and deserialized with gob
		buff := bytes.NewBuffer(nil)
		enc := gob.NewEncoder(buff)
		err := enc.Encode(&t)
		if err != nil {
			panic(fmt.Errorf("encode type: %w", err))
		}
		dec := gob.NewDecoder(buff)
		var out any
		err = dec.Decode(&out)
		if err != nil {
			panic(fmt.Errorf("decode type: %w", err))
		}
	}
	return GobMetaEncoder{}
}

func (GobMetaEncoder) Marshal(meta any) ([]byte, error) {
	w := bytes.NewBuffer(nil)
	enc := gob.NewEncoder(w)
	err := enc.Encode(&meta)
	if err != nil {
		return nil, err
	}
	return w.Bytes(), nil
}

func (GobMetaEncoder) Unmarshal(buff []byte) (any, error) {
	r := bytes.NewBuffer(buff)
	dec := gob.NewDecoder(r)
	var out any
	err := dec.Decode(&out)
	if err != nil {
		return nil, err
	}
	return out, nil
}

type rawResponse struct {
	Status      int
	Request     downloader.Request
	RequestMeta [][]byte
	Url         *url.URL
	Headers     http.Header
	Body        []byte
}

func (r rawResponse) Response(menc MetaEncoder) (*downloader.Response, error) {
	req := &r.Request
	for _, buff := range r.RequestMeta {
		rmeta, err := menc.Unmarshal(buff)
		if err != nil {
			return nil, err
		}
		req.AddMeta(rmeta)
	}
	return downloader.NewResponse(req, r.Status, r.Url, r.Headers, r.Body), nil
}

func newRawResponse(r *downloader.Response, menc MetaEncoder) (rawResponse, error) {
	meta := r.Request().Meta()
	var serialized [][]byte
	for _, e := range meta {
		marshalled, err := menc.Marshal(e)
		if err != nil {
			return rawResponse{}, err
		}
		if len(marshalled) == 0 {
			continue
		}
		serialized = append(serialized, marshalled)
	}
	return rawResponse{
		Status:      r.Status(),
		Request:     *r.Request(),
		RequestMeta: serialized,
		Url:         r.Url(),
		Headers:     r.Headers(),
		Body:        r.RawBody(),
	}, nil
}

// FSReplayStore implements CacheStore with the local filesystem.
type FSReplayStore struct {
	dir  string
	menc MetaEncoder
}

func NewFSReplayStore(dir string, menc MetaEncoder) FSReplayStore {
	if menc == nil {
		panic("a valid implementation of MetaEncoder must be given. use middleware.GobMetaEncoder if basic serialization with encoding/gob is all you need for your use case")
	}
	return FSReplayStore{dir: dir, menc: menc}
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

	res, err := rr.Response(s.menc)
	if err != nil {
		logger.Warn("fs_cache_store", "decode response", "path", path, "err", err)
		return nil
	}
	return res
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

	rawres, err := newRawResponse(res, s.menc)
	if err != nil {
		logger.Warn("fs_cache_store", "encode response", "path", path, "err", err)
		return
	}

	encoder := gob.NewEncoder(f)
	err = encoder.Encode(rawres)
	if err != nil {
		logger.Warn("fs_cache_store", "encode response", "path", path, "err", err)
	}
}
