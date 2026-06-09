package gopdfxt

import (
	"context"
	"errors"
	"testing"
)

func TestNewRejectsMissingLLMOptions(t *testing.T) {
	_, err := New(Options{})
	if !errors.Is(err, ErrInvalidOptions) {
		t.Fatalf("expected ErrInvalidOptions, got %v", err)
	}
}

func TestNewAcceptsMinimalLLMOptions(t *testing.T) {
	converter, err := New(Options{
		LLM: LLMOptions{
			Provider: ProviderQwen,
			APIKey:   "test-key",
			Model:    "qwen3-vl-plus",
		},
		Extractor: fakeExtractor{},
	})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	if converter == nil {
		t.Fatalf("expected converter")
	}
}

func TestNewAcceptsDefaultExtractor(t *testing.T) {
	converter, err := New(Options{
		LLM: LLMOptions{
			Provider: ProviderQwen,
			APIKey:   "test-key",
			Model:    "qwen3-vl-plus",
		},
	})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	if converter == nil {
		t.Fatalf("expected converter")
	}
}

func TestNewDefaultExtractorReturnsExtractor(t *testing.T) {
	var extractor Extractor = NewDefaultExtractor()
	if extractor == nil {
		t.Fatalf("expected default extractor")
	}
}

func TestNewKeepsExplicitExtractor(t *testing.T) {
	extractor := fakeExtractor{}
	converter, err := New(Options{
		LLM: LLMOptions{
			Provider: ProviderQwen,
			APIKey:   "test-key",
			Model:    "qwen3-vl-plus",
		},
		Extractor: extractor,
	})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	if converter.options.Extractor != extractor {
		t.Fatalf("expected explicit extractor to be preserved")
	}
}

type fakeExtractor struct{}

func (fakeExtractor) Extract(ctx context.Context, input PDFInput) (*ExtractedDocument, error) {
	return &ExtractedDocument{PDF: input.Path}, nil
}
