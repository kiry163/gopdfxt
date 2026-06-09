package pipeline

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/kiry163/gopdfxt/internal/document"
	"github.com/kiry163/gopdfxt/internal/render"
)

type Input struct {
	Path string
}

type Result struct {
	Content string
	Articles []Article
}

type Article struct {
	Title   string
	Content string
	Start   int
	End     int
}

type Extractor interface {
	Extract(context.Context, Input) (*document.Document, error)
}

type Classifier interface {
	ClassifyPage(context.Context, document.Page) (document.PageStructure, error)
}

type Options struct {
	Extractor   Extractor
	Classifier  Classifier
	Concurrency int
	Hooks       Hooks
}

type Hooks struct {
	OnConvertStart func(context.Context, ConvertStartEvent)
	OnConvertDone  func(context.Context, ConvertDoneEvent)
	OnPageStart    func(context.Context, PageStartEvent)
	OnPageDone     func(context.Context, PageDoneEvent)
	OnPageError    func(context.Context, PageErrorEvent)
}

type ConvertStartEvent struct{}

type ConvertDoneEvent struct {
	Result  *Result
	Elapsed time.Duration
}

type PageStartEvent struct {
	PageIndex int
}

type PageDoneEvent struct {
	PageIndex int
	Elapsed   time.Duration
}

type PageErrorEvent struct {
	PageIndex int
	Err       error
}

func Convert(ctx context.Context, opts Options, input Input) (*Result, error) {
	started := time.Now()
	if opts.Hooks.OnConvertStart != nil {
		opts.Hooks.OnConvertStart(ctx, ConvertStartEvent{})
	}

	doc, err := opts.Extractor.Extract(ctx, input)
	if err != nil {
		return nil, err
	}

	pageResults, err := classifyPages(ctx, opts, doc.Pages)
	if err != nil {
		return nil, err
	}

	groups := render.BuildGroups(doc, pageResults)
	content := render.RenderContent(doc, groups)
	articles := render.SplitArticles(doc, groups)
	result := &Result{
		Content:  content,
		Articles: toArticles(articles),
	}

	if opts.Hooks.OnConvertDone != nil {
		opts.Hooks.OnConvertDone(ctx, ConvertDoneEvent{Result: result, Elapsed: time.Since(started)})
	}
	return result, nil
}

func classifyPages(ctx context.Context, opts Options, pages []document.Page) ([]document.PageStructure, error) {
	if opts.Classifier == nil {
		return nil, fmt.Errorf("pipeline classifier is required")
	}
	concurrency := opts.Concurrency
	if concurrency <= 0 {
		concurrency = 4
	}

	results := make([]document.PageStructure, len(pages))
	sem := make(chan struct{}, concurrency)
	var wg sync.WaitGroup
	var firstErr error
	var errMu sync.Mutex

	for i, page := range pages {
		if ctx.Err() != nil {
			break
		}
		sem <- struct{}{}
		wg.Add(1)
		go func(i int, page document.Page) {
			defer wg.Done()
			defer func() { <-sem }()

			started := time.Now()
			if opts.Hooks.OnPageStart != nil {
				opts.Hooks.OnPageStart(ctx, PageStartEvent{PageIndex: page.PageIndex})
			}

			result, err := opts.Classifier.ClassifyPage(ctx, page)
			if err != nil {
				if opts.Hooks.OnPageError != nil {
					opts.Hooks.OnPageError(ctx, PageErrorEvent{PageIndex: page.PageIndex, Err: err})
				}
				errMu.Lock()
				if firstErr == nil {
					firstErr = fmt.Errorf("classify page %d: %w", page.PageIndex+1, err)
				}
				errMu.Unlock()
				return
			}

			results[i] = result
			if opts.Hooks.OnPageDone != nil {
				opts.Hooks.OnPageDone(ctx, PageDoneEvent{PageIndex: page.PageIndex, Elapsed: time.Since(started)})
			}
		}(i, page)
	}

	wg.Wait()
	if firstErr != nil {
		return nil, firstErr
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return results, nil
}

func toArticles(articles []render.Article) []Article {
	if len(articles) == 0 {
		return nil
	}

	result := make([]Article, 0, len(articles))
	for _, article := range articles {
		result = append(result, Article{
			Title:   article.Title,
			Content: article.Content,
			Start:   article.Start,
			End:     article.End,
		})
	}
	return result
}
