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

func TestValidateStructureResultReportsInvalidReferenceIssues(t *testing.T) {
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
	if strings.Contains(joined, "遗漏了未处理的 block_id") {
		t.Fatalf("expected omitted blocks to be allowed, got %v", issues)
	}
}

func TestRepairAnalysisResultDropsInvalidIDsAndIgnoresMissingBlocks(t *testing.T) {
	page := document.Page{
		PageIndex: 0,
		Blocks: []document.Block{
			{BlockID: "a"},
			{BlockID: "b"},
			{BlockID: "c"},
		},
	}

	result := RepairAnalysisResult(page, &PageAnalysisResult{
		PageType: "body",
		Groups: []document.Group{
			{Kind: "paragraph", BlockIDs: []string{"a", "unknown", "a"}},
		},
	})

	if result.PageType != "body" {
		t.Fatalf("expected body page, got %+v", result)
	}
	if len(result.IgnoreBlockIDs) != 2 || result.IgnoreBlockIDs[0] != "b" || result.IgnoreBlockIDs[1] != "c" {
		t.Fatalf("expected omitted ids to be ignored, got %+v", result.IgnoreBlockIDs)
	}
	if len(result.Groups) != 1 {
		t.Fatalf("expected only retained group, got %+v", result.Groups)
	}
	if len(result.Groups[0].BlockIDs) != 1 || result.Groups[0].BlockIDs[0] != "a" {
		t.Fatalf("expected invalid and duplicate ids removed, got %+v", result.Groups[0].BlockIDs)
	}
}
