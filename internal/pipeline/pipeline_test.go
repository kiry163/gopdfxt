package pipeline

import (
	"context"
	"errors"
	"testing"

	"github.com/kiry163/gopdfxt/internal/document"
)

func TestPipelineConvertsExtractedDocument(t *testing.T) {
	extractor := fakeExtractor{doc: &document.Document{
		PDF:       "sample.pdf",
		PageCount: 1,
		Pages: []document.Page{{
			PageIndex:   0,
			ImageBase64: "aGVsbG8=",
			Blocks: []document.Block{
				{BlockID: "p000-b000", PageIndex: 0, Text: "Article One"},
				{BlockID: "p000-b001", PageIndex: 0, Text: "Body"},
			},
		}},
	}}
	classifier := fakeClassifier{results: []document.PageStructure{{
		PageType: "body",
		Groups: []document.Group{
			{Kind: "heading", Level: 1, BlockIDs: []string{"p000-b000"}},
			{Kind: "paragraph", BlockIDs: []string{"p000-b001"}},
		},
	}}}

	result, err := Convert(context.Background(), Options{
		Extractor:   extractor,
		Classifier:  classifier,
		Concurrency: 2,
	}, Input{Path: "sample.pdf"})
	if err != nil {
		t.Fatalf("Convert returned error: %v", err)
	}
	if len(result.Articles) != 1 {
		t.Fatalf("expected one article, got %d", len(result.Articles))
	}
	if result.Articles[0].Title != "Article One" || result.Articles[0].Content != "Body\n" {
		t.Fatalf("unexpected article: %+v", result.Articles[0])
	}
}

func TestPipelinePageDoneIncludesPageCount(t *testing.T) {
	extractor := fakeExtractor{doc: &document.Document{
		PDF:       "sample.pdf",
		PageCount: 2,
		Pages: []document.Page{
			{
				PageIndex:   0,
				ImageBase64: "aGVsbG8=",
				Blocks:      []document.Block{{BlockID: "p000-b000", PageIndex: 0, Text: "First"}},
			},
			{
				PageIndex:   1,
				ImageBase64: "aGVsbG8=",
				Blocks:      []document.Block{{BlockID: "p001-b000", PageIndex: 1, Text: "Second"}},
			},
		},
	}}
	classifier := fakeClassifier{results: []document.PageStructure{
		{
			PageType: "body",
			Groups:   []document.Group{{Kind: "paragraph", BlockIDs: []string{"p000-b000"}}},
		},
		{
			PageType: "body",
			Groups:   []document.Group{{Kind: "paragraph", BlockIDs: []string{"p001-b000"}}},
		},
	}}

	var counts []int
	_, err := Convert(context.Background(), Options{
		Extractor:   extractor,
		Classifier:  classifier,
		Concurrency: 1,
		Hooks: Hooks{
			OnPageDone: func(ctx context.Context, e PageDoneEvent) {
				counts = append(counts, e.PageCount)
			},
		},
	}, Input{Path: "sample.pdf"})
	if err != nil {
		t.Fatalf("Convert returned error: %v", err)
	}
	if len(counts) != 2 {
		t.Fatalf("expected 2 page done events, got %d", len(counts))
	}
	for _, count := range counts {
		if count != 2 {
			t.Fatalf("expected page count 2, got %d", count)
		}
	}
}

func TestPipelinePageDoneIncludesModelStats(t *testing.T) {
	extractor := fakeExtractor{doc: &document.Document{
		PDF:       "sample.pdf",
		PageCount: 1,
		Pages: []document.Page{{
			PageIndex:   0,
			ImageBase64: "aGVsbG8=",
			Blocks:      []document.Block{{BlockID: "p000-b000", PageIndex: 0, Text: "Body"}},
		}},
	}}
	classifier := fakeClassifier{results: []document.PageStructure{{
		PageType:   "body",
		ModelCalls: 1,
		Retries:    0,
		Groups:     []document.Group{{Kind: "paragraph", BlockIDs: []string{"p000-b000"}}},
	}}}

	var event PageDoneEvent
	_, err := Convert(context.Background(), Options{
		Extractor:   extractor,
		Classifier:  classifier,
		Concurrency: 1,
		Hooks: Hooks{
			OnPageDone: func(ctx context.Context, e PageDoneEvent) {
				event = e
			},
		},
	}, Input{Path: "sample.pdf"})
	if err != nil {
		t.Fatalf("Convert returned error: %v", err)
	}
	if event.ModelCalls != 1 || event.Retries != 0 {
		t.Fatalf("unexpected stats: %+v", event)
	}
}

func TestPipelineReturnsErrorOnPageFailureByDefault(t *testing.T) {
	extractor := twoPageExtractor()
	classifier := fakeClassifier{
		results: []document.PageStructure{{PageType: "body", Groups: []document.Group{{Kind: "paragraph", BlockIDs: []string{"p000-b000"}}}}},
		errors:  map[int]error{1: errors.New("model failed")},
	}

	_, err := Convert(context.Background(), Options{
		Extractor:   extractor,
		Classifier:  classifier,
		Concurrency: 1,
	}, Input{Path: "sample.pdf"})
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestPipelineAllowsPartialPageFailure(t *testing.T) {
	extractor := twoPageExtractor()
	classifier := fakeClassifier{
		results: []document.PageStructure{{PageType: "body", ModelCalls: 1, Groups: []document.Group{{Kind: "paragraph", BlockIDs: []string{"p000-b000"}}}}},
		errors:  map[int]error{1: errors.New("model failed")},
	}

	result, err := Convert(context.Background(), Options{
		Extractor:    extractor,
		Classifier:   classifier,
		Concurrency:  1,
		AllowPartial: true,
	}, Input{Path: "sample.pdf"})
	if err != nil {
		t.Fatalf("Convert returned error: %v", err)
	}
	if len(result.FailedPages) != 1 {
		t.Fatalf("expected one failed page, got %+v", result.FailedPages)
	}
	if result.FailedPages[0].PageIndex != 1 || result.FailedPages[0].Error == "" {
		t.Fatalf("unexpected failed page: %+v", result.FailedPages[0])
	}
	if result.Details.PageCount != 2 || result.Details.SucceededPages != 1 || result.Details.FailedPages != 1 {
		t.Fatalf("unexpected details: %+v", result.Details)
	}
	if len(result.Articles) != 1 || result.Articles[0].Content != "First\n" {
		t.Fatalf("unexpected articles: %+v", result.Articles)
	}
}

type fakeExtractor struct {
	doc *document.Document
}

func (f fakeExtractor) Extract(ctx context.Context, input Input) (*document.Document, error) {
	return f.doc, nil
}

type fakeClassifier struct {
	results []document.PageStructure
	errors  map[int]error
}

func (f fakeClassifier) ClassifyPage(ctx context.Context, page document.Page) (document.PageStructure, error) {
	if err := f.errors[page.PageIndex]; err != nil {
		return document.PageStructure{}, err
	}
	return f.results[page.PageIndex], nil
}

func twoPageExtractor() fakeExtractor {
	return fakeExtractor{doc: &document.Document{
		PDF:       "sample.pdf",
		PageCount: 2,
		Pages: []document.Page{
			{
				PageIndex:   0,
				ImageBase64: "aGVsbG8=",
				Blocks:      []document.Block{{BlockID: "p000-b000", PageIndex: 0, Text: "First"}},
			},
			{
				PageIndex:   1,
				ImageBase64: "aGVsbG8=",
				Blocks:      []document.Block{{BlockID: "p001-b000", PageIndex: 1, Text: "Second"}},
			},
		},
	}}
}
