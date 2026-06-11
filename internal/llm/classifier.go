package llm

import (
	"context"

	"github.com/kiry163/gopdfxt/internal/document"
	"github.com/kiry163/gopdfxt/internal/prompt"
)

type Options struct {
	OnRetry func(pageIndex int, stage string, attempt int, err error)
}

type Runner interface {
	RunAnalysis(ctx context.Context, page document.Page, prompt string) (*PageAnalysisResult, error)
}

type Classifier struct {
	runner Runner
	opts   Options
}

func NewClassifier(runner Runner, opts Options) *Classifier {
	return &Classifier{runner: runner, opts: opts}
}

func (c *Classifier) ClassifyPage(ctx context.Context, page document.Page) (document.PageStructure, error) {
	analysis, err := c.classifyAnalysis(ctx, page)
	if err != nil {
		return document.PageStructure{}, err
	}
	if analysis.PageType == "non_body" {
		return document.PageStructure{
			PageType:   "non_body",
			ModelCalls: analysis.ModelCalls,
			Retries:    analysis.Retries,
		}, nil
	}

	return document.PageStructure{
		PageType:       analysis.PageType,
		IgnoreBlockIDs: analysis.IgnoreBlockIDs,
		Groups:         analysis.Groups,
		ModelCalls:     analysis.ModelCalls,
		Retries:        analysis.Retries,
	}, nil
}

func (c *Classifier) classifyAnalysis(ctx context.Context, page document.Page) (*PageAnalysisResult, error) {
	result, err := c.runner.RunAnalysis(ctx, page, prompt.BuildAnalysis(page))
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Classifier) emitRetry(pageIndex int, stage string, attempt int, err error) {
	if c.opts.OnRetry != nil {
		c.opts.OnRetry(pageIndex, stage, attempt, err)
	}
}
