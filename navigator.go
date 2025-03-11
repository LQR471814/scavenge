package scavenge

import (
	"bytes"
	"context"
	"fmt"
	"net/url"

	"github.com/LQR471814/scavenge/downloader"
	"github.com/LQR471814/scavenge/items"

	"golang.org/x/net/html"
)

// Navigator contains methods for saving items and following urls typically used in a
// spider's response handler.
type Navigator struct {
	context    context.Context
	scavenger  *Scavenger
	currentUrl *url.URL
}

// Context returns the scraping context.
func (n Navigator) Context() context.Context {
	return n.context
}

// CurrentUrl returns the current url of the navigator.
func (n Navigator) CurrentUrl() *url.URL {
	return n.currentUrl
}

// SaveItem queues the given item for processing.
func (n Navigator) SaveItem(value any) {
	n.scavenger.QueueItem(items.Item{value})
}

// Request queues the given request.
func (n Navigator) Request(req *downloader.Request) {
	n.scavenger.QueueRequest(req, n.currentUrl)
}

// AnchorUrl returns the absolute url referenced by an anchor tag (created by resolving the href
// with the current url as the base url).
func (n Navigator) AnchorUrl(a *html.Node) (*url.URL, error) {
	if a.Data != "a" || a.Type != html.ElementNode {
		return nil, fmt.Errorf("follow anchor: given html node '%s' (type: %d) was not an <a> tag", a.Data, a.Type)
	}

	href := ""
	for _, attr := range a.Attr {
		if attr.Key == "href" {
			href = attr.Val
			break
		}
	}
	if href == "" {
		rendered := bytes.NewBuffer(nil)
		err := html.Render(rendered, a)
		if err != nil {
			rendered.WriteString(fmt.Errorf("html render error: %w", err).Error())
		}
		return nil, fmt.Errorf(
			"follow anchor: could not find valid href on <a> tag '%s'",
			rendered.String(),
		)
	}

	ref, err := url.Parse(href)
	if err != nil {
		return nil, fmt.Errorf("follow anchor: %w", err)
	}
	abs := n.currentUrl.ResolveReference(ref)
	return abs, nil
}

// AnchorRequest returns a GET request to the given url created by resolving the href of the
// given a-tag with the current url as the base url.
func (n Navigator) AnchorRequest(a *html.Node) (*downloader.Request, error) {
	abs, err := n.AnchorUrl(a)
	if err != nil {
		return nil, err
	}
	return downloader.GETRequest(abs), nil
}
