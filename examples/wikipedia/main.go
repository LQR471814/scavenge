package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"scavenge"
	"scavenge/downloader"
	"scavenge/downloader/middleware"
	"scavenge/item"
	"scavenge/item/pipelines"
	"strings"
	"syscall"

	"github.com/PuerkitoBio/goquery"
	"github.com/lmittmann/tint"
)

// Page is the structured data we retrieve from scraping.
type Page struct {
	Name     string   `json:"name"`
	Overview string   `json:"overview"`
	Sections []string `json:"sections"`
}

// WikipediaSpider contains all the logic for deriving structured data from wikipedia and making new requests.
type WikipediaSpider struct{}

func (s WikipediaSpider) StartingRequests() []*downloader.Request {
	return []*downloader.Request{
		downloader.GETRequest(downloader.MustParseUrl("https://en.wikipedia.org/wiki/Quantum_mechanics")),
		downloader.GETRequest(downloader.MustParseUrl("https://en.wikipedia.org/wiki/Wavelet_transform")),
		downloader.GETRequest(downloader.MustParseUrl("https://en.wikipedia.org/wiki/Bellman_equation")),
	}
}

func (s WikipediaSpider) HandleResponse(nav scavenge.Navigator, res *downloader.Response) error {
	root, err := res.HtmlBody()
	if err != nil {
		return err
	}
	doc := goquery.NewDocumentFromNode(root)

	name := doc.Find("#firstHeading").Text()

	overview := ""
	for _, p := range doc.Find("#mw-content-text p").EachIter() {
		overview = strings.Trim(p.Text(), " \n\t")
		if overview != "" {
			break
		}
	}

	var sections []string
	for _, header := range doc.Find("div.mw-heading").EachIter() {
		sections = append(sections, header.Text())
	}

	nav.SaveItem(Page{
		Name:     name,
		Overview: overview,
		Sections: sections,
	})

	for _, a := range doc.Find("a.mw-redirect").EachIter() {
		nav.FollowAnchor(a.Nodes[0])
	}

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
		middleware.NewAllowedDomains([]string{"en.wikipedia.org"}, nil), // only allow requests with host 'en.wikipedia.org'
		middleware.NewDedupe(), // drop duplicate GET requests with the same url
		middleware.NewReplay( // cache responses from wikipedia on the filesystem so we can replay them later (useful for debugging)
			"default",
			middleware.NewFSReplayStore("replay"),
			middleware.ReplayGetRequests,
		),
		middleware.NewThrottle( // automatically throttle responses
			middleware.NewAutoThrottle(),
		),
	)

	// the wikipedia pages we parse will be written to a file called `out.json`
	// with the ExportJson item pipeline.
	_, err := os.Stat("state.bin")
	isResuming := err == nil

	var out *os.File
	if isResuming {
		// don't truncate results if we are resuming scraping
		out, err = os.OpenFile("out.json", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	} else {
		out, err = os.Create("out.json")
	}
	if err != nil {
		logger.Error("main", "open output json file", "err", err)
		os.Exit(1)
	}
	defer out.Close()

	iproc := item.NewProcessor(
		pipelines.NewExportJson(out),
	)

	// create a ctx that will be canceled on Ctrl+C
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// run the scraper
	scavenger := scavenge.NewScavenger(
		dl, iproc, logger,
		scavenge.WithStateStore(
			scavenge.NewFileStateStore("state.bin"),
		),
	)
	scavenger.Run(ctx, WikipediaSpider{})
}
