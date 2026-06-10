package gopdfxt

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/kiry163/easyllm"
)

var ErrInvalidOptions = errors.New("invalid gopdfxt options")

type Options struct {
	LLM         LLMOptions
	Extractor   Extractor
	Concurrency int
	Timeout     time.Duration
	Hooks       Hooks
	LLMHooks    easyllm.Hooks
}

type LLMOptions struct {
	Provider                string
	APIKey                  string
	BaseURL                 string
	Model                   string
	Temperature             *float64
	TopP                    *float64
	MaxTokens               *int
	EnableThinking          *bool
	Timeout                 time.Duration
	StreamFirstEventTimeout time.Duration
	MaxRetries              int
	InitialBackoff          time.Duration
	MaxBackoff              time.Duration
	ExtraBody               map[string]any
}

type Hooks struct {
	OnConvertStart func(context.Context, ConvertStartEvent)
	OnConvertDone  func(context.Context, ConvertDoneEvent)
	OnPageStart    func(context.Context, PageStartEvent)
	OnPageDone     func(context.Context, PageDoneEvent)
	OnPageError    func(context.Context, PageErrorEvent)
	OnRetry        func(context.Context, RetryEvent)
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
	PageCount int
	Elapsed   time.Duration
}

type PageErrorEvent struct {
	PageIndex int
	Err       error
}

type RetryEvent struct {
	PageIndex int
	Stage     string
	Attempt   int
	Err       error
}

func (o Options) validate() error {
	if strings.TrimSpace(o.LLM.Provider) == "" {
		return fmt.Errorf("%w: llm provider is required", ErrInvalidOptions)
	}
	if strings.TrimSpace(o.LLM.APIKey) == "" {
		return fmt.Errorf("%w: llm api key is required", ErrInvalidOptions)
	}
	if strings.TrimSpace(o.LLM.Model) == "" {
		return fmt.Errorf("%w: llm model is required", ErrInvalidOptions)
	}
	if o.Concurrency < 0 {
		return fmt.Errorf("%w: concurrency must be greater than or equal to zero", ErrInvalidOptions)
	}
	return nil
}

func (o Options) withDefaults() Options {
	if o.Extractor == nil {
		o.Extractor = NewDefaultExtractor()
	}
	return o
}

func (o Options) normalizedConcurrency() int {
	if o.Concurrency <= 0 {
		return 4
	}
	return o.Concurrency
}
