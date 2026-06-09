package prompt

import (
	"strings"
	"testing"

	"github.com/kiry163/gopdfxt/internal/document"
)

func TestBuildContentFilterMentionsNonBodyDirectoryPages(t *testing.T) {
	page := document.Page{
		PageIndex: 1,
		Blocks: []document.Block{
			{BlockID: "p001-b000", BlockType: "text", Text: "目录"},
		},
	}

	prompt := BuildContentFilter(page)

	for _, want := range []string{
		"目录页",
		"page_type",
		"ignore_block_ids",
		"p001-b000",
		"如果拿不准整页是否属于 non_body，优先判为 body",
	} {
		if !strings.Contains(prompt, want) {
			t.Fatalf("expected prompt to contain %q, got %q", want, prompt)
		}
	}
}

func TestBuildContentFilterRetryIncludesPreviousOutputAndIssues(t *testing.T) {
	page := document.Page{PageIndex: 1}
	prompt := BuildContentFilterRetry(page, `{"page_type":"body"}`, []string{"invalid page_type"})

	for _, want := range []string{
		`{"page_type":"body"}`,
		"invalid page_type",
		"ignore_block_ids 不能导致当前页没有任何保留 block",
	} {
		if !strings.Contains(prompt, want) {
			t.Fatalf("expected retry prompt to contain %q, got %q", want, prompt)
		}
	}
}
