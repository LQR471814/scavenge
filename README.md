# scavenge

> A modern, modular scraping framework for go, based on python's scrapy.

## requirements

- [x] method to manage session / cookies
- [x] scrapy spider interface, collect information (items) and continue scraping all in one function
- [x] methods to deduplicate requests
- [x] methods to schedule requests, do rate limiting
- [x] methods to transform scraped export information 
- [ ] convenient method to develop and test scraping code

## design?

### spider (interface)

- `StartingRequests() []scavenge.Request`

creates starting requests.

- `HandleResponse(navigator scavenge.Navigator, response scavenge.Response) error`

`scavenge.Navigator` contains methods for starting new requests, saving an item, and telemetry.

`scavenge.Response` contains response headers, response url, the starting request, a copy of the body, selectors, content-type detection, etc... it also includes context.

`scavenge.Item` is an array of any structs.

Spider produces requests and items.

### downloader

downloader is effectively an HTTP client, you can configure HTTP middleware, cookies, and session headers on this.

### item pipeline

- `NewPipeline(next pipeline...) (handleItem func(tel scavenge.Telemetry, item scavenge.Item) error)`

an item pipeline is effectively middleware for retrieved items. (that means you can have item pipelines chained together, or one pipeline calls multiple other pipelines)
they can do virtually anything with retrieved items (even have side effects).

item pipelines should attempt to cast the item (starting with the first any struct) to an interface that they export, the first struct that fulfills the interface, can be operated on.

to add data to the struct, they can simply add their unique key to the item with their struct as the value.

### telemetry (interface)

an interface containing common logging and trace reporters. it depends on the implementation, what is done with such logs and traces.

### scavenger

the main logic of the scraping, takes in a spider and a downloader.

sends requests to the downloader, brings it back to the spider.

