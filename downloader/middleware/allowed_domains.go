package middleware

import (
	"context"
	"fmt"

	"github.com/LQR471814/scavenge/downloader"

	"github.com/gobwas/glob"
)

// AllowedDomains is a [downloader.DownloaderMiddleware] that limits the domains of requests and responses.
type AllowedDomains struct {
	requestDomains  []glob.Glob
	responseDomains []glob.Glob
}

// NewAllowedDomains creates an AllowedDomains middleware.
//
//   - if forRequests is nil or empty, it will allow all requests.
//   - if forResponses is nil or empty, it will allow all responses.
//
// You can use wildcards (*) in the domains. [documentation](https://github.com/gobwas/glob)
func NewAllowedDomains(forRequests, forResponses []string) AllowedDomains {
	requestDomains := make([]glob.Glob, len(forRequests))
	responseDomains := make([]glob.Glob, len(forResponses))
	for i, d := range forRequests {
		requestDomains[i] = glob.MustCompile(d, '.')
	}
	for i, d := range forResponses {
		responseDomains[i] = glob.MustCompile(d, '.')
	}

	return AllowedDomains{
		requestDomains:  requestDomains,
		responseDomains: responseDomains,
	}
}

func (p AllowedDomains) HandleRequest(ctx context.Context, req *downloader.Request, meta downloader.RequestMetadata) (*downloader.Response, error) {
	if len(p.requestDomains) == 0 {
		return nil, nil
	}

	hostname := req.Url.Hostname()
	matched := false
	for _, domain := range p.requestDomains {
		matched = domain.Match(hostname)
		if matched {
			break
		}
	}
	if !matched {
		return nil, downloader.DroppedRequest(fmt.Errorf(
			"allowed domains: aborting request to '%s', domain '%s' is not allowed",
			req.Url.String(),
			hostname,
		))
	}

	return nil, nil
}

func (p AllowedDomains) HandleResponse(
	ctx context.Context,
	res *downloader.Response,
	meta downloader.ResponseMetadata,
) error {
	if len(p.responseDomains) == 0 {
		return nil
	}

	resUrl := res.Url()
	hostname := resUrl.Hostname()

	matched := false
	for _, domain := range p.requestDomains {
		matched = domain.Match(hostname)
		if matched {
			break
		}
	}
	if !matched {
		return downloader.DroppedRequest(fmt.Errorf(
			"allowed domains: response from domain (for a request to '%s') '%s' is not allowed",
			res.Request().Url,
			hostname,
		))
	}

	return nil
}
