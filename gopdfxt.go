package gopdfxt

import (
	"context"
	"io"
	"os"

	"github.com/kiry163/easyllm"
	"github.com/kiry163/gopdfxt/internal/document"
	internalllm "github.com/kiry163/gopdfxt/internal/llm"
	"github.com/kiry163/gopdfxt/internal/pipeline"
)

type Converter struct {
	options Options
}

func New(opts Options) (*Converter, error) {
	if err := opts.validate(); err != nil {
		return nil, err
	}
	return &Converter{options: opts.withDefaults()}, nil
}

func (c *Converter) ConvertFile(ctx context.Context, path string) (*Result, error) {
	if c.options.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, c.options.Timeout)
		defer cancel()
	}

	client, err := easyllm.NewClient(c.easyllmConfig())
	if err != nil {
		return nil, err
	}
	classifier := internalllm.NewClassifier(internalllm.NewEngineRunner(client, c.options.LLMHooks, c.options.normalizedImageDetail()), internalllm.Options{
		OnRetry: func(pageIndex int, stage string, attempt int, err error) {
			if c.options.Hooks.OnRetry != nil {
				c.options.Hooks.OnRetry(ctx, RetryEvent{
					PageIndex: pageIndex,
					Stage:     stage,
					Attempt:   attempt,
					Err:       err,
				})
			}
		},
	})

	result, err := pipeline.Convert(ctx, pipeline.Options{
		Extractor:    publicExtractorAdapter{extractor: c.options.Extractor},
		Classifier:   classifier,
		Concurrency:  c.options.normalizedConcurrency(),
		AllowPartial: c.options.AllowPartial,
		Hooks:        c.pipelineHooks(),
	}, pipeline.Input{Path: path})
	if err != nil {
		return nil, err
	}
	return publicResult(result), nil
}

func (c *Converter) ConvertReader(ctx context.Context, r io.Reader) (*Result, error) {
	tmp, err := os.CreateTemp("", "gopdfxt-*.pdf")
	if err != nil {
		return nil, err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)

	if _, err := io.Copy(tmp, r); err != nil {
		tmp.Close()
		return nil, err
	}
	if err := tmp.Close(); err != nil {
		return nil, err
	}
	return c.ConvertFile(ctx, tmpPath)
}

func (c *Converter) easyllmConfig() easyllm.Config {
	return easyllm.Config{
		Provider:                c.options.LLM.Provider,
		APIKey:                  c.options.LLM.APIKey,
		BaseURL:                 c.options.LLM.BaseURL,
		Model:                   c.options.LLM.Model,
		Temperature:             c.options.LLM.Temperature,
		TopP:                    c.options.LLM.TopP,
		MaxTokens:               c.options.LLM.MaxTokens,
		EnableThinking:          c.options.LLM.EnableThinking,
		Timeout:                 c.options.LLM.Timeout,
		StreamFirstEventTimeout: c.options.LLM.StreamFirstEventTimeout,
		MaxRetries:              c.options.LLM.MaxRetries,
		InitialBackoff:          c.options.LLM.InitialBackoff,
		MaxBackoff:              c.options.LLM.MaxBackoff,
		ExtraBody:               c.options.LLM.ExtraBody,
	}
}

func (c *Converter) pipelineHooks() pipeline.Hooks {
	return pipeline.Hooks{
		OnConvertStart: func(ctx context.Context, e pipeline.ConvertStartEvent) {
			if c.options.Hooks.OnConvertStart != nil {
				c.options.Hooks.OnConvertStart(ctx, ConvertStartEvent{})
			}
		},
		OnConvertDone: func(ctx context.Context, e pipeline.ConvertDoneEvent) {
			if c.options.Hooks.OnConvertDone != nil {
				c.options.Hooks.OnConvertDone(ctx, ConvertDoneEvent{
					Result:  publicResult(e.Result),
					Elapsed: e.Elapsed,
				})
			}
		},
		OnPageStart: func(ctx context.Context, e pipeline.PageStartEvent) {
			if c.options.Hooks.OnPageStart != nil {
				c.options.Hooks.OnPageStart(ctx, PageStartEvent{PageIndex: e.PageIndex})
			}
		},
		OnPageDone: func(ctx context.Context, e pipeline.PageDoneEvent) {
			if c.options.Hooks.OnPageDone != nil {
				c.options.Hooks.OnPageDone(ctx, PageDoneEvent{
					PageIndex:  e.PageIndex,
					PageCount:  e.PageCount,
					ModelCalls: e.ModelCalls,
					Retries:    e.Retries,
					Elapsed:    e.Elapsed,
				})
			}
		},
		OnPageError: func(ctx context.Context, e pipeline.PageErrorEvent) {
			if c.options.Hooks.OnPageError != nil {
				c.options.Hooks.OnPageError(ctx, PageErrorEvent{PageIndex: e.PageIndex, Err: e.Err})
			}
		},
	}
}

type publicExtractorAdapter struct {
	extractor Extractor
}

func (a publicExtractorAdapter) Extract(ctx context.Context, input pipeline.Input) (*document.Document, error) {
	doc, err := a.extractor.Extract(ctx, PDFInput{Path: input.Path})
	if err != nil {
		return nil, err
	}
	return internalDocument(doc), nil
}

func internalDocument(doc *ExtractedDocument) *document.Document {
	if doc == nil {
		return nil
	}
	result := &document.Document{
		PDF:       doc.PDF,
		PageCount: doc.PageCount,
		Pages:     make([]document.Page, 0, len(doc.Pages)),
	}
	for _, page := range doc.Pages {
		internalPage := document.Page{
			PageIndex:   page.PageIndex,
			Width:       page.Width,
			Height:      page.Height,
			ImageBase64: page.ImageBase64,
			Blocks:      make([]document.Block, 0, len(page.Blocks)),
		}
		for _, block := range page.Blocks {
			internalPage.Blocks = append(internalPage.Blocks, document.Block{
				BlockID:   block.BlockID,
				PageIndex: block.PageIndex,
				BlockType: block.BlockType,
				Text:      block.Text,
				BBox:      append([]float64(nil), block.BBox...),
				FontSize:  block.FontSize,
				Fonts:     append([]string(nil), block.Fonts...),
				LineCount: block.LineCount,
			})
		}
		result.Pages = append(result.Pages, internalPage)
	}
	return result
}

func publicResult(result *pipeline.Result) *Result {
	if result == nil {
		return nil
	}
	return &Result{
		Articles:    publicArticles(result.Articles),
		FailedPages: publicPageErrors(result.FailedPages),
		Details: ProcessingDetails{
			PageCount:      result.Details.PageCount,
			SucceededPages: result.Details.SucceededPages,
			FailedPages:    result.Details.FailedPages,
			ModelCalls:     result.Details.ModelCalls,
			Retries:        result.Details.Retries,
		},
	}
}

func publicArticles(articles []pipeline.Article) []Article {
	if len(articles) == 0 {
		return nil
	}
	result := make([]Article, 0, len(articles))
	for _, article := range articles {
		result = append(result, Article{
			Title:   article.Title,
			Content: article.Content,
			Pages: PageRange{
				Start: article.Start,
				End:   article.End,
			},
		})
	}
	return result
}

func publicPageErrors(pages []pipeline.PageError) []PageError {
	if len(pages) == 0 {
		return nil
	}
	result := make([]PageError, 0, len(pages))
	for _, page := range pages {
		result = append(result, PageError{
			PageIndex: page.PageIndex,
			Error:     page.Error,
		})
	}
	return result
}
