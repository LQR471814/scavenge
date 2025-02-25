package middleware

import (
	"context"
	"net/http"
	"net/http/cookiejar"

	"github.com/LQR471814/scavenge"
	"github.com/LQR471814/scavenge/downloader"
)

// Cookies persists cookies across requests.
type Cookies struct {
	jar *cookiejar.Jar
}

func NewCookies(jar *cookiejar.Jar) Cookies {
	return Cookies{
		jar: jar,
	}
}

func (c Cookies) HandleRequest(ctx context.Context, req *downloader.Request, meta downloader.RequestMetadata) (*downloader.Response, error) {
	cookies := c.jar.Cookies(req.Url)
	for _, c := range cookies {
		req.Headers.Add("cookie", c.String())
	}
	return nil, nil
}

func (c Cookies) HandleResponse(ctx context.Context, res *downloader.Response, meta downloader.ResponseMetadata) error {
	logger := scavenge.LoggerFromContext(ctx)

	for k, values := range res.Headers() {
		if k != "Cookie" {
			continue
		}
		cookies := make([]*http.Cookie, 0, len(values))
		for _, v := range values {
			c, err := http.ParseSetCookie(v)
			if err != nil {
				logger.Warn("cookies", "parse set-cookie", "err", err)
				continue
			}
			cookies = append(cookies, c)
		}
		c.jar.SetCookies(res.Url(), cookies)
	}

	return nil
}
