package main

import (
	"context"
	"fmt"
	"scavenge/downloader"
	"sync/atomic"
)

type MaxRequests struct {
	MaxCount uint64
	count    atomic.Uint64
}

func (m *MaxRequests) HandleRequest(ctx context.Context, req *downloader.Request, meta downloader.RequestMetadata) (*downloader.Response, error) {
	if m.count.Add(1) > m.MaxCount {
		return nil, downloader.DroppedRequest(fmt.Errorf("max request count exceeded"))
	}
	return nil, nil
}

func (m *MaxRequests) HandleResponse(ctx context.Context, res *downloader.Response, meta downloader.ResponseMetadata) error {
	return nil
}
