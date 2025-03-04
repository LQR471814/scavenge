package scavenge

import (
	"bytes"
	"context"
	"fmt"
	"net/url"

	"github.com/LQR471814/scavenge/downloader"
	"github.com/LQR471814/scavenge/item"

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

// SaveItem queues the given item for processing.
func (n Navigator) SaveItem(value any) {
	n.scavenger.QueueItem(item.Item{value})
}

// Request queues the given request.
func (n Navigator) Request(req *downloader.Request) {
	n.scavenger.QueueRequest(req, n.currentUrl)
}

// FollowUrl queues a GET request to the given url with the current url as the referer.
//
// Note: This will not resolve the given url with the current url as the base url.
func (n Navigator) FollowUrl(u *url.URL) {
	n.scavenger.QueueRequest(downloader.GETRequest(u), n.currentUrl)
}

// FollowAnchor queues a GET request to the given url created by resolving the href of the
// given a-tag with the current url as the base url.
func (n Navigator) FollowAnchor(a *html.Node) error {
	if a.Data != "a" || a.Type != html.ElementNode {
		return fmt.Errorf("follow anchor: given html node '%s' (type: %d) was not an <a> tag", a.Data, a.Type)
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
		return fmt.Errorf(
			"follow anchor: could not find valid href on <a> tag '%s'",
			rendered.String(),
		)
	}

	ref, err := url.Parse(href)
	if err != nil {
		return fmt.Errorf("follow anchor: %w", err)
	}
	abs := n.currentUrl.ResolveReference(ref)

	n.scavenger.QueueRequest(downloader.GETRequest(abs), n.currentUrl)

	return nil
}
