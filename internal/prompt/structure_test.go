package prompt

import (
	"strings"
	"testing"

	"github.com/kiry163/gopdfxt/internal/document"
)

func TestBuildAnalysisCombinesFilteringAndStructure(t *testing.T) {
	page := document.Page{
		PageIndex: 0,
		Blocks: []document.Block{
			{BlockID: "p000-b000", BlockType: "text", Text: "Title"},
		},
	}

	prompt := BuildAnalysis(page)

	for _, want := range []string{
		"submit_page_analysis",
		"page_type",
		"start_block_id",
		"end_block_id",
		"未出现在 groups 中的 block 会被忽略",
		"groups",
		"heading",
		"paragraph",
		"p000-b000",
	} {
		if !strings.Contains(prompt, want) {
			t.Fatalf("expected prompt to contain %q, got %q", want, prompt)
		}
	}
	if strings.Contains(prompt, "ignore_block_ids") {
		t.Fatalf("expected prompt not to contain ignore_block_ids, got %q", prompt)
	}
}
