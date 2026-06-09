package pipeline

import (
	"context"
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

type fakeExtractor struct {
	doc *document.Document
}

func (f fakeExtractor) Extract(ctx context.Context, input Input) (*document.Document, error) {
	return f.doc, nil
}

type fakeClassifier struct {
	results []document.PageStructure
}

func (f fakeClassifier) ClassifyPage(ctx context.Context, page document.Page) (document.PageStructure, error) {
	return f.results[page.PageIndex], nil
}
