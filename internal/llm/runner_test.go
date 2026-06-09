package llm

import (
	"context"
	"testing"

	"github.com/kiry163/easyllm"
	"github.com/kiry163/gopdfxt/internal/document"
)

func TestEngineRunnerRunFilterUsesToolCall(t *testing.T) {
	client := toolClient{
		toolName: "submit_page_filter",
		args: map[string]any{
			"page_type":        "body",
			"ignore_block_ids": []any{"p000-b001", "p000-b001"},
		},
	}
	runner := NewEngineRunner(client, easyllm.Hooks{})

	result, err := runner.RunFilter(context.Background(), document.Page{
		PageIndex: 0,
		Blocks: []document.Block{
			{BlockID: "p000-b000"},
			{BlockID: "p000-b001"},
		},
	}, "filter prompt")
	if err != nil {
		t.Fatalf("RunFilter returned error: %v", err)
	}
	if result.PageType != "body" {
		t.Fatalf("expected body page, got %+v", result)
	}
	if len(result.IgnoreBlockIDs) != 1 || result.IgnoreBlockIDs[0] != "p000-b001" {
		t.Fatalf("expected deduped ignore ids, got %+v", result.IgnoreBlockIDs)
	}
}

func TestEngineRunnerRunStructureUsesToolCall(t *testing.T) {
	client := toolClient{
		toolName: "submit_page_structure",
		args: map[string]any{
			"groups": []any{
				map[string]any{
					"kind":      "heading",
					"level":     1,
					"block_ids": []any{"p000-b000"},
				},
			},
		},
	}
	runner := NewEngineRunner(client, easyllm.Hooks{})

	result, err := runner.RunStructure(context.Background(), document.Page{
		PageIndex: 0,
		Blocks: []document.Block{
			{BlockID: "p000-b000"},
		},
	}, "structure prompt")
	if err != nil {
		t.Fatalf("RunStructure returned error: %v", err)
	}
	if len(result.Groups) != 1 || result.Groups[0].Kind != "heading" {
		t.Fatalf("unexpected structure result: %+v", result)
	}
}

type toolClient struct {
	toolName string
	args     map[string]any
}

func (c toolClient) Generate(ctx context.Context, req easyllm.ModelRequest) (*easyllm.ModelResponse, error) {
	return &easyllm.ModelResponse{
		Output: []easyllm.OutputItem{
			easyllm.AssistantOutput{
				Role: "assistant",
				ToolCalls: []easyllm.ToolCallOutput{
					{
						CallID:    "call_1",
						Name:      c.toolName,
						Arguments: c.args,
					},
				},
			},
		},
		FinishReason: "tool_calls",
	}, nil
}

func (c toolClient) GenerateStream(ctx context.Context, req easyllm.ModelRequest, handler easyllm.StreamHandler) error {
	panic("not implemented")
}
