package llm

import (
	"context"
	"testing"

	"github.com/kiry163/gopdfxt/internal/document"
)

func TestClassifierClassifiesBodyPage(t *testing.T) {
	runner := &fakeToolRunner{
		filter: PageFilterResult{PageType: "body"},
		structure: StructureResult{Groups: []document.Group{
			{Kind: "heading", Level: 1, BlockIDs: []string{"p000-b000"}},
		}},
	}
	classifier := NewClassifier(runner, Options{})
	page := document.Page{
		PageIndex:   0,
		ImageBase64: "aGVsbG8=",
		Blocks: []document.Block{
			{BlockID: "p000-b000", PageIndex: 0, Text: "Title"},
		},
	}

	got, err := classifier.ClassifyPage(context.Background(), page)
	if err != nil {
		t.Fatalf("ClassifyPage returned error: %v", err)
	}
	if got.PageType != "body" || len(got.Groups) != 1 {
		t.Fatalf("unexpected page structure: %+v", got)
	}
}

func TestClassifierReturnsNonBodyWithoutStructureCall(t *testing.T) {
	runner := &fakeToolRunner{
		filter: PageFilterResult{PageType: "non_body"},
	}
	classifier := NewClassifier(runner, Options{})

	got, err := classifier.ClassifyPage(context.Background(), document.Page{PageIndex: 0})
	if err != nil {
		t.Fatalf("ClassifyPage returned error: %v", err)
	}
	if got.PageType != "non_body" {
		t.Fatalf("expected non_body page, got %+v", got)
	}
	if runner.structureCalls != 0 {
		t.Fatalf("expected no structure tool call, got %d", runner.structureCalls)
	}
}

type fakeToolRunner struct {
	filter         PageFilterResult
	structure      StructureResult
	filterCalls    int
	structureCalls int
}

func (f *fakeToolRunner) RunFilter(ctx context.Context, page document.Page, prompt string) (*PageFilterResult, error) {
	f.filterCalls++
	return &f.filter, nil
}

func (f *fakeToolRunner) RunStructure(ctx context.Context, page document.Page, prompt string) (*StructureResult, error) {
	f.structureCalls++
	return &f.structure, nil
}
