package llm

import (
	"strings"
	"testing"

	"github.com/kiry163/gopdfxt/internal/document"
)

func TestValidatePageFilterResultRejectsBodyWithNoKeptBlocks(t *testing.T) {
	page := document.Page{
		PageIndex: 0,
		Blocks: []document.Block{
			{BlockID: "a"},
			{BlockID: "b"},
		},
	}

	issues := ValidatePageFilterResult(page, &PageFilterResult{
		PageType:       "body",
		IgnoreBlockIDs: []string{"a", "b"},
	})
	if len(issues) == 0 {
		t.Fatalf("expected issues for empty kept blocks")
	}
	if !strings.Contains(strings.Join(issues, "\n"), "non_body") {
		t.Fatalf("expected non_body guidance, got %v", issues)
	}
}

func TestValidateStructureResultReportsCoverageIssues(t *testing.T) {
	page := document.Page{
		PageIndex: 0,
		Blocks: []document.Block{
			{BlockID: "a"},
			{BlockID: "b"},
		},
	}

	result := &StructureResult{
		Groups: []document.Group{
			{Kind: "paragraph", BlockIDs: []string{"a", "c"}},
			{Kind: "paragraph", BlockIDs: []string{"a"}},
		},
	}

	issues := ValidateStructureResult(page, result)
	joined := strings.Join(issues, "\n")
	if !strings.Contains(joined, "重复出现在") {
		t.Fatalf("expected duplicate issue, got %v", issues)
	}
	if !strings.Contains(joined, "未知 block_id") {
		t.Fatalf("expected unknown id issue, got %v", issues)
	}
	if !strings.Contains(joined, "遗漏了未处理的 block_id: b") {
		t.Fatalf("expected missing id issue, got %v", issues)
	}
}
