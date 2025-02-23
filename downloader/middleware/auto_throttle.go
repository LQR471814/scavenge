package middleware

import (
	"context"
	"fmt"
	"scavenge/downloader"
	"sync"
	"time"
)

type ThrottleHandler interface {
	// Throttle returns an integer that represents the amount of time to wait before making the request.
	Throttle(ctx context.Context, req *downloader.Request) (delay time.Duration)
	// HandleResponse will be called with all the responses returned by the HTTP client.
	HandleResponse(ctx context.Context, dl downloader.Downloader, res *downloader.Response, meta downloader.ResponseMetadata)
}

// Throttle throttles crawling speed to ease load on website servers.
//
// Note: This is partly based on scrapy's [AutoThrottle](https://docs.scrapy.org/en/latest/topics/autothrottle.html) extension.
type Throttle struct {
	handler ThrottleHandler
}

func NewThrottle(handler ThrottleHandler) Throttle {
	return Throttle{handler: handler}
}

func (t Throttle) HandleRequest(ctx context.Context, dl downloader.Downloader, req *downloader.Request) (*downloader.Response, error) {
	delay := t.handler.Throttle(ctx, req)
	time.Sleep(delay)
	return nil, nil
}

func (t Throttle) HandleResponse(
	ctx context.Context,
	dl downloader.Downloader,
	res *downloader.Response,
	meta downloader.ResponseMetadata,
) error {
	t.handler.HandleResponse(ctx, dl, res, meta)
	return nil
}

type autoThrottleCfg struct {
	startDelay        time.Duration
	minDelay          time.Duration
	maxDelay          time.Duration
	targetConcurrency int
}

type autoThrottleOption = func(cfg *autoThrottleCfg)

func WithAutoThrottleStartDelay(delay time.Duration) autoThrottleOption {
	return func(cfg *autoThrottleCfg) {
		cfg.startDelay = delay
	}
}

func WithAutoThrottleDelayBounds(minDelay, maxDelay time.Duration) autoThrottleOption {
	return func(cfg *autoThrottleCfg) {
		if minDelay > maxDelay {
			panic(fmt.Errorf("auto throttle: min delay '%v' cannot be greater than max delay '%v'", minDelay, maxDelay))
		}
		cfg.minDelay = minDelay
		cfg.maxDelay = maxDelay
	}
}

func WithAutoThrottleTargetConcurrency(concurrency int) autoThrottleOption {
	return func(cfg *autoThrottleCfg) {
		cfg.targetConcurrency = concurrency
	}
}

// AutoThrottle automatically limits scraping speed in order to lessen the burden on websites, avoid rate-limiting, and decrease overall scraping time.
//
//   - Based on scrapy's AutoThrottle [algorithm](https://docs.scrapy.org/en/latest/topics/autothrottle.html#throttling-algorithm).
type AutoThrottle struct {
	cfg  autoThrottleCfg
	dict sync.Map
}

func NewAutoThrottle(options ...autoThrottleOption) *AutoThrottle {
	cfg := autoThrottleCfg{
		startDelay:        0,
		minDelay:          0,
		maxDelay:          time.Minute,
		targetConcurrency: 1,
	}
	for _, o := range options {
		o(&cfg)
	}
	return &AutoThrottle{
		cfg:  cfg,
		dict: sync.Map{},
	}
}

func (a *AutoThrottle) getTargetDelay(req *downloader.Request) time.Duration {
	v, ok := a.dict.Load(req.Url.Host)
	delay := v.(time.Duration)
	if !ok {
		delay = a.cfg.startDelay
	}
	tdelay := delay / time.Duration(a.cfg.targetConcurrency)
	if tdelay > a.cfg.maxDelay {
		return a.cfg.maxDelay
	}
	if tdelay < a.cfg.minDelay {
		return a.cfg.minDelay
	}
	return tdelay
}

func (a *AutoThrottle) Throttle(ctx context.Context, req *downloader.Request) time.Duration {
	return a.getTargetDelay(req)
}

func (a *AutoThrottle) HandleResponse(ctx context.Context, dl downloader.Downloader, res *downloader.Response, meta downloader.ResponseMetadata) {
	if res.Status() != 200 {
		return
	}
	prevTargetDelay := a.getTargetDelay(res.Request())
	a.dict.Store(res.Url().Host, (prevTargetDelay+meta.Elapsed)/2)
}
