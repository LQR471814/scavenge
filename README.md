# scavenge

> A modern, batteries-included, modular scraping framework for Go.

## Features

- **Minimal footprint:** ~1k LOC with minimal reliance on third-party libraries.
- **Batteries-included:** Domain whitelisting, automatic throttling, replaying responses, etc... all included.
- **Flexible:** Near-full control over the details of scraping, but no need to concern yourself with the details if you don't need to.
- **Extensible:** Easily add features you need with simple middleware interfaces.

## Example

Here's an [example](./examples/wikipedia) that scrapes 1000 wikipedia pages in ~10 seconds.

To run it:

```sh
cd examples/wikipedia
go run .
```

## Dependencies

- `golang.org/x/net` - Used in `downloader.Response`.
- `github.com/PuerkitoBio/purell` - Used only in `middleware.Dedupe` and `middleware.Replay`.
- `github.com/gobwas/glob` - Used only in `middleware.AllowedDomains`.
- `github.com/zeebo/xxh3` - Used only in `middleware.FSReplayStore`.
- All the other dependencies are only used in examples.

## Credits

- Python's [scrapy](https://scrapy.org/) is a massive influence on this library, many of the design choices in this library are taken straight from here.

