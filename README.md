# gopdfxt

`gopdfxt` is a Go library for converting academic PDF content into structured article content with VLM assistance.

Model calls are powered by [easyllm](https://github.com/kiry163/easyllm). PDF extraction is built in.

## Install

```bash
go get github.com/kiry163/gopdfxt
```

## Quick Start

```go
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/kiry163/gopdfxt"
)

func main() {
	converter, err := gopdfxt.New(gopdfxt.Options{
		LLM: gopdfxt.LLMOptions{
			Provider: gopdfxt.ProviderQwen,
			APIKey:   os.Getenv("DASHSCOPE_API_KEY"),
			Model:    "qwen3-vl-plus",
		},
	})
	if err != nil {
		panic(err)
	}

	result, err := converter.ConvertFile(context.Background(), "paper.pdf")
	if err != nil {
		panic(err)
	}

	for _, article := range result.Articles {
		fmt.Println(article.Title)
		fmt.Println(article.Content)
	}
}
```

## Result

```go
type Result struct {
	Articles []Article
}

type Article struct {
	Title   string
	Content string
	Pages   PageRange
}
```

`Articles` contains article-level splits when level-one headings are identified. `Content` is plain text output; it does not include heading markers like `#`.

## LLM Options

The root package re-exports easyllm provider constants:

```go
gopdfxt.ProviderOpenAI
gopdfxt.ProviderQwen
gopdfxt.ProviderDeepSeek
gopdfxt.ProviderOpenAICompatible
```

Common model settings live in `LLMOptions`:

```go
converter, err := gopdfxt.New(gopdfxt.Options{
	LLM: gopdfxt.LLMOptions{
		Provider: gopdfxt.ProviderOpenAICompatible,
		BaseURL:  "https://example.com/v1",
		APIKey:   os.Getenv("MODEL_API_KEY"),
		Model:    "your-vlm-model",
		ImageDetail: gopdfxt.ImageDetailLow,
		Timeout:  2 * time.Minute,
	},
	Concurrency: 8,
	AllowPartial: true,
})
```

Set `AllowPartial` to keep successful pages when some pages fail. Failed pages are returned in `Result.FailedPages`, and aggregate counts are returned in `Result.Details`.

Use `DefaultExtractor` when you want to tune the built-in PDF extractor:

```go
converter, err := gopdfxt.New(gopdfxt.Options{
	LLM:       llmOptions,
	Extractor: &gopdfxt.DefaultExtractor{DPI: 192},
})
```

## Hooks

Use hooks for progress and observability. Hook failures do not fail conversion.

```go
converter, err := gopdfxt.New(gopdfxt.Options{
	LLM: llmOptions,
	Hooks: gopdfxt.Hooks{
		OnPageDone: func(ctx context.Context, e gopdfxt.PageDoneEvent) {
			log.Printf("page %d finished in %s", e.PageIndex, e.Elapsed)
		},
		OnRetry: func(ctx context.Context, e gopdfxt.RetryEvent) {
			log.Printf("page %d retry stage=%s attempt=%d err=%v", e.PageIndex, e.Stage, e.Attempt, e.Err)
		},
	},
})
```

For lower-level easyllm lifecycle events, pass `LLMHooks`:

```go
converter, err := gopdfxt.New(gopdfxt.Options{
	LLM: llmOptions,
	LLMHooks: easyllm.Hooks{
		OnModelRequest: func(e easyllm.ModelRequestEvent) error {
			log.Println("model request")
			return nil
		},
	},
})
```

The built-in extractor is used by default. Pass `Extractor` only if you want to override it or tune its `DPI`.

## CLI Example

The repository includes a small example command:

```bash
go run ./cmd/gopdfxt -input paper.pdf -api-key "$DASHSCOPE_API_KEY" -result result.json
```

CLI model network requests retry 3 times by default after the first attempt. Set `-max-retries 0` to disable retries, or pass another value to tune it:

```bash
go run ./cmd/gopdfxt -input paper.pdf -api-key "$DASHSCOPE_API_KEY" -max-retries 5
```

Each LLM network request has a 120 second timeout by default. The CLI does not set an overall conversion timeout, so request timeouts can retry without cancelling the whole conversion:

```bash
go run ./cmd/gopdfxt -input paper.pdf -api-key "$DASHSCOPE_API_KEY" -llm-timeout 3m
```

The command is intentionally thin and intended for local testing and debugging; library usage is the primary API.
It writes a JSON result file by default, including extracted article content, failed pages, and processing details.
