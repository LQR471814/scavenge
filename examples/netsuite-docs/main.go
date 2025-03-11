package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	htmltomarkdown "github.com/JohannesKaufmann/html-to-markdown/v2"
	"github.com/JohannesKaufmann/html-to-markdown/v2/converter"
	"github.com/LQR471814/scavenge"
	"github.com/LQR471814/scavenge/downloader"
	"github.com/LQR471814/scavenge/downloader/middleware"
	"github.com/LQR471814/scavenge/items"
	"github.com/LQR471814/scavenge/items/pipelines"
	"golang.org/x/net/html"

	"github.com/PuerkitoBio/goquery"
	"github.com/lmittmann/tint"
)

// Page is the structured data we retrieve from scraping.
type Page struct {
	Name     string `json:"name"`
	Url      string `json:"url"`
	Markdown string `json:"markdown"`

	content *html.Node
}

// Spider contains all the logic for deriving structured data from wikipedia and making new requests.
type Spider struct{}

func (s Spider) StartingRequests() []*downloader.Request {
	req := downloader.GETRequest(downloader.MustParseUrl(
		"https://docs.oracle.com/en/cloud/saas/netsuite/ns-online-help/toc.htm",
	))
	return []*downloader.Request{req}
}

func (s Spider) HandleResponse(nav scavenge.Navigator, res *downloader.Response) error {
	root, err := res.HtmlBody()
	if err != nil {
		return err
	}
	doc := goquery.NewDocumentFromNode(root)

	if strings.HasSuffix(res.Url().Path, "toc.htm") {
		for _, a := range doc.Find("a").EachIter() {
			if len(a.Nodes) == 0 {
				continue
			}
			req, err := nav.AnchorRequest(a.Nodes[0])
			if err != nil {
				return err
			}
			nav.Request(req)
		}
		return nil
	}

	url := res.Url().String()

	title := doc.Find("header h1")
	article := doc.Find("article")
	if len(article.Nodes) == 0 {
		return nil
	}

	nav.SaveItem(Page{
		Name:    title.Text(),
		Url:     url,
		content: article.Nodes[0],
	})

	return nil
}

func main() {
	// pretty logging
	slogger := slog.New(tint.NewHandler(
		os.Stderr,
		&tint.Options{
			Level: slog.LevelDebug,
		},
	))
	logger := scavenge.NewSlogLogger(slogger, false)

	// creates a new downloader that wraps an http client in some middleware
	dl := downloader.NewDownloader(
		downloader.NewHttpClient(http.DefaultClient),

		// middleware is evaluated from top to bottom for each request
		middleware.NewAllowedDomains([]string{"docs.oracle.com"}, nil), // only allow requests with host 'en.wikipedia.org'
		middleware.NewDedupe(), // drop duplicate GET requests with the same url
		middleware.NewReplay( // cache responses from wikipedia on the filesystem so we can replay them later (useful for debugging)
			"netsuite-docs",
			middleware.NewFSReplayStore("replay", middleware.NewGobMetaEncoder()),
			middleware.ReplayGetRequests,
		),
		middleware.NewThrottle( // automatically throttle responses
			middleware.NewAutoThrottle(),
		),
	)

	var out *os.File
	out, err := os.Create("out.json")
	if err != nil {
		logger.Error("main", "open output json file", "err", err)
		os.Exit(1)
	}
	defer out.Close()

	iproc := items.NewProcessor(
		ConvertPipeline{},
		pipelines.NewExportJson(out),
	)

	// create a ctx that will be canceled on Ctrl+C
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// run the scraper
	scavenger := scavenge.NewScavenger(dl, iproc, logger)
	scavenger.Run(ctx, Spider{})
}

type ConvertPipeline struct{}

func (c ConvertPipeline) HandleItem(ctx context.Context, item items.Item) (items.Item, error) {
	p, ok := items.CastItem[Page](item)
	if !ok {
		return nil, fmt.Errorf("invalid item")
	}

	md, err := htmltomarkdown.ConvertNode(
		p.content,
		converter.WithDomain(p.Url),
	)
	if err != nil {
		return nil, err
	}
	p.Markdown = string(md)

	return items.Item{p}, nil
}
