package scavenge

import (
	"bytes"
	"fmt"
	"net/url"
	"scavenge/downloader"
	"scavenge/item"

	"golang.org/x/net/html"
)

// Navigator contains methods for saving items and following urls typically used in a
// spider's response handler.
type Navigator struct {
	scavenger  *Scavenger
	currentUrl *url.URL
}

func (n Navigator) SaveItem(value any) {
	n.scavenger.queueItemJob(item.Item{value})
}

func (n Navigator) Request(req *downloader.Request) {
	n.scavenger.queueReqJob(req, n.currentUrl)
}

func (n Navigator) FollowUrl(u *url.URL) {
	n.scavenger.queueReqJob(downloader.GETRequest(u), n.currentUrl)
}

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

	n.scavenger.reqjobs <- reqJob{
		req:     downloader.GETRequest(abs),
		referer: n.currentUrl,
	}
	return nil
}
