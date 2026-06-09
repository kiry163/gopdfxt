package render

import (
	"testing"

	"github.com/kiry163/gopdfxt/internal/document"
)

func TestBuildGroupsMergesParagraphAcrossPages(t *testing.T) {
	doc := &document.Document{
		PageCount: 2,
		Pages: []document.Page{
			{
				PageIndex: 0,
				Width:     600,
				Height:    800,
				Blocks: []document.Block{
					{
						BlockID:   "p000-b000",
						PageIndex: 0,
						BlockType: "text",
						Text:      "First paragraph continues",
						BBox:      []float64{0, 700, 100, 790},
						FontSize:  12,
					},
				},
			},
			{
				PageIndex: 1,
				Width:     600,
				Height:    800,
				Blocks: []document.Block{
					{
						BlockID:   "p001-b000",
						PageIndex: 1,
						BlockType: "text",
						Text:      "on the next page",
						BBox:      []float64{0, 10, 100, 60},
						FontSize:  12,
					},
				},
			},
		},
	}

	pageResults := []document.PageStructure{
		{
			PageType: "body",
			Groups: []document.Group{
				{Kind: "paragraph", BlockIDs: []string{"p000-b000"}},
			},
		},
		{
			PageType: "body",
			Groups: []document.Group{
				{Kind: "paragraph", BlockIDs: []string{"p001-b000"}},
			},
		},
	}

	groups := BuildGroups(doc, pageResults)
	if len(groups) != 1 {
		t.Fatalf("expected 1 merged group, got %d", len(groups))
	}

	got := RenderContent(doc, groups)
	want := "First paragraph continues on the next page\n"
	if got != want {
		t.Fatalf("expected content %q, got %q", want, got)
	}
}

func TestBuildGroupsNormalizesNumberedHeading(t *testing.T) {
	doc := &document.Document{
		PageCount: 1,
		Pages: []document.Page{
			{
				PageIndex: 0,
				Blocks: []document.Block{
					{
						BlockID:   "p000-b000",
						PageIndex: 0,
						BlockType: "text",
						Text:      "1. 绪论",
						BBox:      []float64{0, 0, 100, 30},
						FontSize:  18,
					},
				},
			},
		},
	}

	pageResults := []document.PageStructure{
		{
			PageType: "body",
			Groups: []document.Group{
				{Kind: "heading", Level: 1, BlockIDs: []string{"p000-b000"}},
			},
		},
	}

	groups := BuildGroups(doc, pageResults)
	if len(groups) != 1 {
		t.Fatalf("expected 1 group, got %d", len(groups))
	}
	if groups[0].Kind != "paragraph" {
		t.Fatalf("expected numbered heading to normalize to paragraph, got %s", groups[0].Kind)
	}
}
