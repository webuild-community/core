# go-notion

[![GitHub tag (latest SemVer)](https://img.shields.io/github/v/tag/dstotijn/go-notion?label=go%20module)](https://github.com/dstotijn/go-notion/tags)
[![Go Reference](https://pkg.go.dev/badge/github.com/dstotijn/go-notion.svg)](https://pkg.go.dev/github.com/dstotijn/go-notion)
[![GitHub](https://img.shields.io/github/license/dstotijn/go-notion)](LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/dstotijn/go-notion)](https://goreportcard.com/report/github.com/dstotijn/go-notion)

**go-notion** is a client for the
[Notion API](https://developers.notion.com/reference), written in
[Go](https://golang.org/).

## Features

The client supports all (non-deprecated) endpoints available in the Notion API,
as of May 15, 2021:

- [x] [Retrieve a database](https://pkg.go.dev/github.com/dstotijn/go-notion#Client.FindDatabaseByID)
- [x] [Query a database](https://pkg.go.dev/github.com/dstotijn/go-notion#Client.QueryDatabase)
- [x] [Retrieve a page](https://pkg.go.dev/github.com/dstotijn/go-notion#Client.FindPageByID)
- [x] [Create a page](https://pkg.go.dev/github.com/dstotijn/go-notion#Client.CreatePage)
- [x] [Update page properties](https://pkg.go.dev/github.com/dstotijn/go-notion#Client.UpdatePageProps)
- [x] [Retrieve block children](https://pkg.go.dev/github.com/dstotijn/go-notion#Client.FindBlockChildrenByID)
- [x] [Append block children](https://pkg.go.dev/github.com/dstotijn/go-notion#Client.AppendBlockChildren)
- [x] [Retrieve a user](https://pkg.go.dev/github.com/dstotijn/go-notion#Client.FindUserByID)
- [x] [List all users](https://pkg.go.dev/github.com/dstotijn/go-notion#Client.ListUsers)
- [x] [Search](https://pkg.go.dev/github.com/dstotijn/go-notion#Client.Search)

## Installation

```sh
$ go get github.com/dstotijn/go-notion
```

## Getting started

To obtain an API key, follow Notion’s [getting started guide](https://developers.notion.com/docs/getting-started).

### Code example

First, construct a new `Client`:

```go
import "github.com/dstotijn/go-notion"

(...)

client := notion.NewClient("secret-api-key")
```

Then, use the methods defined on `Client` to make requests to the API. For
example:

```go
page, err := client.FindPageByID(context.Background(), "18d35eb5-91f1-4dcb-85b0-c340fd965015")
if err != nil {
    // Handle error...
}
```

👉 Check out the docs on
[pkg.go.dev](https://pkg.go.dev/github.com/dstotijn/go-notion) for further
reference and examples.

## Status

The Notion API is currently in _public beta_.

⚠️ Although the API itself is versioned, this client **will** make breaking
changes in its code until `v1.0` of the module is released.

### To do

- [ ] Write tests
- [ ] Provide examples

## License

[MIT License](LICENSE)

---

© 2021 David Stotijn — [Twitter](https://twitter.com/dstotijn),
[Email](mailto:dstotijn@gmail.com)
