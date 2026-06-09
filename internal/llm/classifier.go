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
	RunFilter(ctx context.Context, page document.Page, prompt string) (*PageFilterResult, error)
	RunStructure(ctx context.Context, page document.Page, prompt string) (*StructureResult, error)
}

type Classifier struct {
	runner Runner
	opts   Options
}

func NewClassifier(runner Runner, opts Options) *Classifier {
	return &Classifier{runner: runner, opts: opts}
}

func (c *Classifier) ClassifyPage(ctx context.Context, page document.Page) (document.PageStructure, error) {
	filter, err := c.classifyFilter(ctx, page)
	if err != nil {
		return document.PageStructure{}, err
	}
	if filter.PageType == "non_body" {
		return document.PageStructure{PageType: "non_body"}, nil
	}

	filtered := FilterPageBlocks(page, filter)
	structure, err := c.classifyStructure(ctx, filtered)
	if err != nil {
		return document.PageStructure{}, err
	}

	return document.PageStructure{
		PageType:       "body",
		IgnoreBlockIDs: filter.IgnoreBlockIDs,
		Groups:         structure.Groups,
	}, nil
}

func (c *Classifier) classifyFilter(ctx context.Context, page document.Page) (*PageFilterResult, error) {
	result, err := c.runner.RunFilter(ctx, page, prompt.BuildContentFilter(page))
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Classifier) classifyStructure(ctx context.Context, page document.Page) (*StructureResult, error) {
	result, err := c.runner.RunStructure(ctx, page, prompt.BuildStructure(page))
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
