package scavenge

import (
	"context"
	"encoding/gob"
	"fmt"
	"math"
	"math/rand/v2"
	"net/url"
	"runtime"
	"scavenge/downloader"
	"scavenge/item"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// Spider contains the business logic of navigating to different links and storing structured data from
// the unstructured page.
//
// Note: HandleResponse should be safe to be called concurrently, if it is too hard to make it so,
// make sure that any scavenger that uses this spider has the option WithDownloadWorkerCount
// set to 1.
type Spider interface {
	StartingRequests() []*downloader.Request
	HandleResponse(nav Navigator, res *downloader.Response) error
}

// Scavenger
type Scavenger struct {
	cfg   config
	log   Logger
	dl    downloader.Downloader
	iproc item.Processor

	reqjobs     chan reqJob
	itemjobs    chan item.Item
	wg          sync.WaitGroup
	queued      atomic.Int64
	quitWorkers atomic.Uint64
}

type config struct {
	stateStore        StateStore
	parallelDownloads int
	parallelItems     int
	minRetryDelay     time.Duration
	maxRetryDelay     time.Duration
	reqFailHandler    func(req *downloader.Request, err error)
	spiderFailHandler func(res *downloader.Response, err error)
	iprocFailHandler  func(i item.Item, err error)
}

type option func(cfg *config)

// WithStateStore sets the interface the Scavenger uses for saving incomplete scraping state.
func WithStateStore(store StateStore) option {
	return func(cfg *config) {
		cfg.stateStore = store
	}
}

// WithRetryDelayBounds sets the bounds for retry delay.
func WithRetryDelayBounds(minDelay, maxDelay time.Duration) option {
	return func(cfg *config) {
		if minDelay > maxDelay {
			panic(fmt.Errorf(
				"min retry delay '%v' was greater than max retry delay '%v'",
				minDelay,
				maxDelay,
			))
		}
		cfg.minRetryDelay = minDelay
		cfg.maxRetryDelay = maxDelay
	}
}

// WithParallelDownloads sets the amount of requests and responses that can be processed in parallel.
func WithParallelDownloads(count int) option {
	return func(cfg *config) {
		cfg.parallelDownloads = count
	}
}

// WithParallelItems sets the amount of items that can be processed in parallel.
func WithParallelItems(count int) option {
	return func(cfg *config) {
		cfg.parallelItems = count
	}
}

// WithOnRequestFail sets the error handler for requests that:
//
//   - fail before sending, either due to a malformed request or failed middleware.
//   - fail while sending, due to some http transport error.
//   - fail after receiving a response that causes an error in the downloader's middleware.
func WithOnRequestFail(fn func(req *downloader.Request, err error)) option {
	return func(cfg *config) {
		cfg.reqFailHandler = fn
	}
}

// WithOnSpiderHandlerFail sets the error handler for errors that occur within
// [Spider.HandleResponse]
func WithOnSpiderHandlerFail(callback func(res *downloader.Response, err error)) option {
	return func(cfg *config) {
		cfg.spiderFailHandler = callback
	}
}

// WithOnItemProcessorFail sets the error handler for item processing errors.
func WithOnItemProcessorFail(callback func(i item.Item, err error)) option {
	return func(cfg *config) {
		cfg.iprocFailHandler = callback
	}
}

// NewScavenger creates a new Scavenger.
func NewScavenger(
	dl downloader.Downloader,
	items item.Processor,
	logger Logger,
	options ...option,
) *Scavenger {
	defaultParDowns := runtime.NumCPU() / 2
	cfg := config{
		parallelDownloads: defaultParDowns,
		parallelItems:     runtime.NumCPU() - defaultParDowns,
	}
	for _, opt := range options {
		opt(&cfg)
	}
	return &Scavenger{
		cfg:   cfg,
		log:   logger,
		dl:    dl,
		iproc: items,
	}
}

func (s *Scavenger) handleRequest(
	ctx context.Context,
	spider Spider,
	job reqJob,
) {
	defer s.wg.Done()

	s.log.Info(
		"scavenger", "download",
		"url", ShortUrl(job.Req.Url),
		"referer", ShortUrl(job.Referer),
		"attempt", job.attempt,
	)

	ctx = setLogCtx(ctx, s.log)

	res, err := s.dl.Download(ctx, job.Req, downloader.RequestMetadata{
		AttemptNo: job.attempt,
		Referer:   job.Referer,
	})
	if err != nil {
		if strings.Contains(err.Error(), "dropped request:") {
			s.log.Info(
				"scavenger", "dropped request",
				"url", ShortUrl(job.Req.Url),
				"referer", ShortUrl(job.Referer),
				"attempt", job.attempt,
				"err", err,
			)
			return
		}

		s.log.Error(
			"scavenger", "request download failed",
			"url", ShortUrl(job.Req.Url),
			"referer", ShortUrl(job.Referer),
			"attempt", job.attempt,
			"err", err,
		)
		if s.cfg.reqFailHandler != nil {
			s.cfg.reqFailHandler(job.Req, err)
		}
		s.retryReqJob(ctx, job)
		return
	}

	err = spider.HandleResponse(Navigator{
		context:    ctx,
		scavenger:  s,
		currentUrl: res.Url(),
	}, res)
	if err != nil {
		err := fmt.Errorf("spider: %w", err)
		s.log.Error(
			"scavenger", "spider handle response failed",
			"url", ShortUrl(job.Req.Url),
			"referer", ShortUrl(job.Referer),
			"attempt", job.attempt,
			"err", err,
		)
		if s.cfg.spiderFailHandler != nil {
			s.cfg.spiderFailHandler(res, err)
		}
		s.retryReqJob(ctx, job)
	}
}

func (s *Scavenger) handleItem(i item.Item) {
	defer s.wg.Done()

	_, err := s.iproc.Process(i)
	if err != nil {
		s.log.Error(
			"scavenger", "item processing failed",
			"item", i,
			"err", err,
		)
		if s.cfg.iprocFailHandler != nil {
			s.cfg.iprocFailHandler(i, err)
		}
		return
	}
}

func (s *Scavenger) loadState() (resumed bool) {
	if s.cfg.stateStore == nil {
		return false
	}
	r, err := s.cfg.stateStore.Load()
	if err != nil {
		s.log.Error("scavenger", "load state: open store for reading", "err", err)
		return false
	}

	decoder := gob.NewDecoder(r)
	var remaining state
	err = decoder.Decode(&remaining)
	if err != nil {
		s.log.Error("scavenger", "load state: decode state", "err", err)
		return false
	}

	for _, reqjob := range remaining.reqs {
		s.queueReqJob(reqjob.Req, reqjob.Referer)
	}
	for _, itemjob := range remaining.items {
		s.queueItemJob(itemjob)
	}

	return len(remaining.reqs) > 0 || len(remaining.items) > 0
}

// this function can only be called if there is more than one on the waitgroup
func (s *Scavenger) saveState() {
	totalWorkers := uint64(s.cfg.parallelDownloads) + uint64(s.cfg.parallelItems)

	// wait until all the workers have exited to start flushing channels, we can be certain
	// that all writes to channels at this point have been blocked
	if s.quitWorkers.Add(1) < totalWorkers {
		return
	}

	s.wg.Add(1)
	defer s.wg.Done()

	s.log.Info("scavenger", "shutting down...")

	var remaining state
	for s.queued.Load() > 0 {
		select {
		case job := <-s.reqjobs:
			s.wg.Done()
			remaining.reqs = append(remaining.reqs, job)
		case item := <-s.itemjobs:
			s.wg.Done()
			remaining.items = append(remaining.items, item)
		default:
		}
	}

	s.log.Info(
		"scavenger", "flushed state successfully",
		"pending_requests", len(remaining.reqs),
		"pending_items", len(remaining.items),
	)

	if s.cfg.stateStore == nil {
		s.log.Info("scavenger", "shutdown successful")
		return
	}
	w, err := s.cfg.stateStore.Store()
	if err != nil {
		s.log.Error("scavenger", "save state: open store for writing", "err", err)
		return
	}
	defer w.Close()

	encoder := gob.NewEncoder(w)
	err = encoder.Encode(remaining)
	if err != nil {
		s.log.Error("scavenger", "save state: encode state", "err", err)
		return
	}

	s.log.Info("scavenger", "state saved successfully")
}

func (s *Scavenger) queueReqJob(req *downloader.Request, referer *url.URL) {
	s.wg.Add(1)
	s.queued.Add(1)
	go func() {
		defer s.queued.Add(-1)

		s.reqjobs <- reqJob{
			Req:     req,
			Referer: referer,
		}
	}()
}

func (s *Scavenger) retryReqJob(ctx context.Context, reqjob reqJob) {
	s.wg.Add(1)
	s.queued.Add(1)
	go func() {
		defer s.queued.Add(-1)

		reqjob.attempt++

		seconds := int(math.Pow(2, float64(reqjob.attempt)))
		jitter := rand.IntN(seconds)
		delay := time.Second * time.Duration(seconds+jitter)
		timer := time.NewTimer(delay)

		for {
			select {
			case <-ctx.Done():
				timer.Stop()
				s.reqjobs <- reqjob
				return
			case <-timer.C:
				timer.Stop()
				s.reqjobs <- reqjob
				return
			}
		}
	}()
}

func (s *Scavenger) queueItemJob(i item.Item) {
	s.wg.Add(1)
	s.queued.Add(1)
	go func() {
		defer s.queued.Add(-1)

		s.itemjobs <- i
	}()
}

func (s *Scavenger) reqWorker(ctx context.Context, spider Spider) {
	for {
		select {
		case <-ctx.Done():
			s.saveState()
			return
		case job := <-s.reqjobs:
			s.handleRequest(ctx, spider, job)
		}
	}
}

func (s *Scavenger) itemWorker(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			s.saveState()
			return
		case item := <-s.itemjobs:
			s.handleItem(item)
		}
	}
}

// Run runs the given spider on the scavenger, this should not be run concurrently.
func (s *Scavenger) Run(ctx context.Context, spider Spider) {
	s.log.Info(
		"scavenger", "running spider",
		"download_workers", s.cfg.parallelDownloads,
		"item_workers", s.cfg.parallelItems,
	)

	s.itemjobs = make(chan item.Item, 256)
	s.reqjobs = make(chan reqJob, 256)
	s.wg = sync.WaitGroup{}

	for range s.cfg.parallelDownloads {
		go s.reqWorker(ctx, spider)
	}
	for range s.cfg.parallelItems {
		go s.itemWorker(ctx)
	}

	resumed := s.loadState()
	if resumed {
		s.log.Info("scavenger", "resuming from saved state...")
	} else {
		requests := spider.StartingRequests()
		for _, r := range requests {
			s.queueReqJob(r, nil)
		}
	}

	s.wg.Wait()
}
