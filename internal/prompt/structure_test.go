package prompt

import (
	"strings"
	"testing"

	"github.com/kiry163/gopdfxt/internal/document"
)

func TestBuildStructureRequiresCoverage(t *testing.T) {
	page := document.Page{
		PageIndex: 0,
		Blocks: []document.Block{
			{BlockID: "p000-b000", BlockType: "text", Text: "Title"},
		},
	}

	prompt := BuildStructure(page)

	for _, want := range []string{
		"每个 block_id 必须且只能出现一次",
		"groups",
		"heading",
		"paragraph",
		"p000-b000",
	} {
		if !strings.Contains(prompt, want) {
			t.Fatalf("expected prompt to contain %q, got %q", want, prompt)
		}
	}
}
