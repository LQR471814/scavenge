package scavenge

import (
	"context"
	"fmt"
	"net/url"
	"runtime"
	"scavenge/downloader"
	"scavenge/item"
	"sync"
)

// Spider contains the business logic of navigating to different links and storing structured data from
// the unstructured page.
//
// Note: HandleResponse should be safe to be called concurrently, if it is too hard to make it so,
// make sure that any scavenger that uses this spider has the option WithDownloadWorkerCount
// set to 1.
type Spider interface {
	StartingRequests() []*downloader.Request
	HandleResponse(navigator Navigator, response *downloader.Response) error
}

// Scavenger
type Scavenger struct {
	cfg   config
	log   Logger
	dl    downloader.Downloader
	iproc item.Processor

	reqjobs  chan reqJob
	itemjobs chan item.Item
	wg       sync.WaitGroup
}

type reqJob struct {
	req     *downloader.Request
	referer *url.URL
}

type config struct {
	parallelDownloads int
	parallelItems     int
	reqFailHandler    func(req *downloader.Request, err error)
	spiderFailHandler func(res *downloader.Response, err error)
	iprocFailHandler  func(i item.Item, err error)
}

type option func(cfg *config)

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
	tel Logger,
	options ...option,
) *Scavenger {
	cfg := config{
		parallelDownloads: runtime.NumCPU(),
	}
	for _, opt := range options {
		opt(&cfg)
	}
	return &Scavenger{
		cfg:   cfg,
		log:   tel,
		dl:    dl,
		iproc: items,
	}
}

func (s *Scavenger) queueReqJob(req *downloader.Request, referer *url.URL) {
	s.wg.Add(1)
	s.reqjobs <- reqJob{
		req:     req,
		referer: referer,
	}
}

func (s *Scavenger) queueItemJob(i item.Item) {
	s.wg.Add(1)
	s.itemjobs <- i
}

func (s *Scavenger) handleRequest(
	ctx context.Context,
	spider Spider,
	job reqJob,
) {
	defer s.wg.Done()

	s.log.Info("scavenger", "download", "url", job.req.Url, "referer", job.referer.String())

	res, err := s.dl.Download(ctx, job.req)
	if err != nil {
		err := fmt.Errorf("downloader: %w", err)
		s.log.Error(
			"scavenger", "request download failed",
			"url", job.req.Url,
			"referer", job.referer,
			"error", err,
		)
		if s.cfg.reqFailHandler != nil {
			s.cfg.reqFailHandler(job.req, err)
		}
		return
	}

	err = spider.HandleResponse(Navigator{
		scavenger:  s,
		currentUrl: res.Url(),
	}, res)
	if err != nil {
		err := fmt.Errorf("spider: %w", err)
		s.log.Error(
			"scavenger", "spider handle response failed",
			"url", job.req.Url,
			"referer", job.referer,
			"error", err,
		)
		if s.cfg.spiderFailHandler != nil {
			s.cfg.spiderFailHandler(res, err)
		}
	}
}

func (s *Scavenger) reqWorker(ctx context.Context, spider Spider) {
	for {
		select {
		case job := <-s.reqjobs:
			s.handleRequest(ctx, spider, job)
		case <-ctx.Done():
			return
		}
	}
}

func (s *Scavenger) handleItem(i item.Item) {
	defer s.wg.Done()

	_, err := s.iproc.Process(i)
	if err != nil {
		s.log.Error(
			"scavenger", "item processing failed",
			"item", i,
			"error", err,
		)
		if s.cfg.iprocFailHandler != nil {
			s.cfg.iprocFailHandler(i, err)
		}
		return
	}
}

func (s *Scavenger) itemWorker(ctx context.Context) {
	for {
		select {
		case item := <-s.itemjobs:
			s.handleItem(item)
		case <-ctx.Done():
			return
		}
	}
}

// Run runs the given spider on the scavenger, this should not be run concurrently.
func (s *Scavenger) Run(ctx context.Context, spider Spider) {
	s.log.Info("scavenger", "running spider", "workers", s.cfg.parallelDownloads)

	s.itemjobs = make(chan item.Item)
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
