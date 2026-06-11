package llm

import (
	"context"
	"testing"

	"github.com/kiry163/gopdfxt/internal/document"
)

func TestClassifierClassifiesBodyPage(t *testing.T) {
	runner := &fakeToolRunner{
		analysis: PageAnalysisResult{PageType: "body", Groups: []document.Group{
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
	if runner.analysisCalls != 1 {
		t.Fatalf("expected one analysis call, got %d", runner.analysisCalls)
	}
}

func TestClassifierReturnsNonBodyFromSingleAnalysisCall(t *testing.T) {
	runner := &fakeToolRunner{
		analysis: PageAnalysisResult{PageType: "non_body"},
	}
	classifier := NewClassifier(runner, Options{})

	got, err := classifier.ClassifyPage(context.Background(), document.Page{PageIndex: 0})
	if err != nil {
		t.Fatalf("ClassifyPage returned error: %v", err)
	}
	if got.PageType != "non_body" {
		t.Fatalf("expected non_body page, got %+v", got)
	}
	if runner.analysisCalls != 1 {
		t.Fatalf("expected one analysis call, got %d", runner.analysisCalls)
	}
}

type fakeToolRunner struct {
	analysis      PageAnalysisResult
	analysisCalls int
}

func (f *fakeToolRunner) RunAnalysis(ctx context.Context, page document.Page, prompt string) (*PageAnalysisResult, error) {
	f.analysisCalls++
	return &f.analysis, nil
}
