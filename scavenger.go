package scavenge

import (
	"context"
	"fmt"
	"math"
	"math/rand/v2"
	"net/url"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/LQR471814/scavenge/downloader"
	"github.com/LQR471814/scavenge/item"
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

// Scavenger is the main component that schedules requests and processes items concurrently,
// it handles retry logic and pausing/resuming scraping.
type Scavenger struct {
	cfg   config
	log   Logger
	dl    downloader.Downloader
	iproc item.Processor

	reqjobs     chan reqJob
	itemjobs    chan itemJob
	wg          sync.WaitGroup
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
		minRetryDelay:     time.Second,
		maxRetryDelay:     time.Hour,
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

func (s *Scavenger) handleItem(ctx context.Context, job itemJob) {
	defer s.wg.Done()

	_, err := s.iproc.Process(job.Item)
	if err != nil {
		s.log.Error(
			"scavenger", "item processing failed",
			"item", job.Item,
			"err", err,
		)
		if s.cfg.iprocFailHandler != nil {
			s.cfg.iprocFailHandler(job.Item, err)
		}
		s.retryItemJob(ctx, job)
		return
	}
}

// this function can only be called if there is more than one on the waitgroup
func (s *Scavenger) flushState() {
	totalWorkers := uint64(s.cfg.parallelDownloads) + uint64(s.cfg.parallelItems)

	// wait until all the workers have exited to start flushing channels, we can be certain
	// that all writes to channels at this point have been blocked
	if s.quitWorkers.Add(1) < totalWorkers {
		return
	}

	close(s.reqjobs)
	close(s.itemjobs)

	s.log.Info("scavenger", "shutdown successful")
}

func (s *Scavenger) recoverAndCancelJob() {
	err := recover()
	if err != nil {
		s.wg.Done()
	}
}

func (s *Scavenger) queueReqJob(req *downloader.Request, referer *url.URL) {
	s.wg.Add(1)
	go func() {
		defer s.recoverAndCancelJob()

		s.reqjobs <- reqJob{
			Req:     req,
			Referer: referer,
		}
	}()
}

func (s *Scavenger) queueItemJob(i item.Item) {
	s.wg.Add(1)
	go func() {
		defer s.recoverAndCancelJob()

		s.itemjobs <- itemJob{Item: i}
	}()
}

func (s *Scavenger) retryDelay(attempt int) time.Duration {
	seconds := int(math.Pow(2, float64(attempt)))
	jitter := rand.IntN(seconds)
	delay := time.Second * time.Duration(seconds+jitter)
	if delay < s.cfg.minRetryDelay {
		return s.cfg.minRetryDelay
	}
	if delay > s.cfg.maxRetryDelay {
		return s.cfg.maxRetryDelay
	}
	return delay
}

func (s *Scavenger) retryReqJob(ctx context.Context, job reqJob) {
	s.wg.Add(1)
	go func() {
		defer s.recoverAndCancelJob()

		job.attempt++
		timer := time.NewTimer(s.retryDelay(job.attempt))
		defer timer.Stop()

		for {
			select {
			case <-ctx.Done():
				s.wg.Done()
				return
			case <-timer.C:
				s.reqjobs <- job
				return
			}
		}
	}()
}

func (s *Scavenger) retryItemJob(ctx context.Context, job itemJob) {
	s.wg.Add(1)
	go func() {
		defer s.recoverAndCancelJob()

		job.attempt++
		timer := time.NewTimer(s.retryDelay(job.attempt))
		defer timer.Stop()

		for {
			select {
			case <-ctx.Done():
				s.wg.Done()
				return
			case <-timer.C:
				s.itemjobs <- job
				return
			}
		}
	}()
}

func (s *Scavenger) reqWorker(ctx context.Context, spider Spider) {
	for {
		select {
		case <-ctx.Done():
			s.flushState()
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
			s.flushState()
			return
		case item := <-s.itemjobs:
			s.handleItem(ctx, item)
		}
	}
}

// Run runs the given spider on the scavenger.
//
// Note:
//   - Run is not concurrency-safe, it should only be executed one-at-a-time for a given Scavenger.
func (s *Scavenger) Run(ctx context.Context, spider Spider) {
	s.log.Info(
		"scavenger", "running spider",
		"download_workers", s.cfg.parallelDownloads,
		"item_workers", s.cfg.parallelItems,
	)

	s.itemjobs = make(chan itemJob)
	s.reqjobs = make(chan reqJob)
	s.wg = sync.WaitGroup{}

	for range s.cfg.parallelDownloads {
		go s.reqWorker(ctx, spider)
	}
	for range s.cfg.parallelItems {
		go s.itemWorker(ctx)
	}

	requests := spider.StartingRequests()
	for _, r := range requests {
		s.queueReqJob(r, nil)
	}

	s.wg.Wait()
}
