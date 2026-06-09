package render

import (
	"testing"

	"github.com/kiry163/gopdfxt/internal/document"
)

func TestSplitArticlesByHeadingLevelOne(t *testing.T) {
	doc := &document.Document{
		PDF:       "/tmp/sample.pdf",
		PageCount: 3,
		Pages: []document.Page{
			{
				PageIndex: 0,
				Blocks: []document.Block{
					{BlockID: "p000-b000", PageIndex: 0, Text: "Article One"},
					{BlockID: "p000-b001", PageIndex: 0, Text: "First body"},
				},
			},
			{
				PageIndex: 1,
				Blocks: []document.Block{
					{BlockID: "p001-b000", PageIndex: 1, Text: "Continued body"},
				},
			},
			{
				PageIndex: 2,
				Blocks: []document.Block{
					{BlockID: "p002-b000", PageIndex: 2, Text: "Article Two"},
					{BlockID: "p002-b001", PageIndex: 2, Text: "Second body"},
				},
			},
		},
	}

	groups := []document.Group{
		{Kind: "heading", Level: 1, BlockIDs: []string{"p000-b000"}},
		{Kind: "paragraph", BlockIDs: []string{"p000-b001", "p001-b000"}},
		{Kind: "heading", Level: 1, BlockIDs: []string{"p002-b000"}},
		{Kind: "paragraph", BlockIDs: []string{"p002-b001"}},
	}

	articles := SplitArticles(doc, groups)
	if len(articles) != 2 {
		t.Fatalf("expected 2 articles, got %d", len(articles))
	}

	if articles[0].Title != "Article One" {
		t.Fatalf("expected first article title %q, got %q", "Article One", articles[0].Title)
	}
	if articles[0].Start != 0 || articles[0].End != 1 {
		t.Fatalf("expected first article range 0-1, got %d-%d", articles[0].Start, articles[0].End)
	}
	if articles[0].Content != "First body Continued body\n" {
		t.Fatalf("unexpected first article content: %q", articles[0].Content)
	}

	if articles[1].Title != "Article Two" {
		t.Fatalf("expected second article title %q, got %q", "Article Two", articles[1].Title)
	}
	if articles[1].Start != 2 || articles[1].End != 2 {
		t.Fatalf("expected second article range 2-2, got %d-%d", articles[1].Start, articles[1].End)
	}
}

func TestSplitArticlesFallsBackToSingleDocument(t *testing.T) {
	doc := &document.Document{
		PDF:       "/tmp/sample.pdf",
		PageCount: 2,
		Pages: []document.Page{
			{
				PageIndex: 0,
				Blocks: []document.Block{
					{BlockID: "p000-b000", PageIndex: 0, Text: "Intro"},
				},
			},
			{
				PageIndex: 1,
				Blocks: []document.Block{
					{BlockID: "p001-b000", PageIndex: 1, Text: "Body"},
				},
			},
		},
	}

	groups := []document.Group{
		{Kind: "paragraph", BlockIDs: []string{"p000-b000", "p001-b000"}},
	}

	articles := SplitArticles(doc, groups)
	if len(articles) != 1 {
		t.Fatalf("expected 1 article, got %d", len(articles))
	}
	if articles[0].Title != "sample" {
		t.Fatalf("expected fallback title %q, got %q", "sample", articles[0].Title)
	}
	if articles[0].Start != 0 || articles[0].End != 1 {
		t.Fatalf("expected fallback range 0-1, got %d-%d", articles[0].Start, articles[0].End)
	}
	if articles[0].Content != "Intro Body\n" {
		t.Fatalf("unexpected fallback content: %q", articles[0].Content)
	}
}
