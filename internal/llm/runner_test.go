package llm

import (
	"context"
	"testing"

	"github.com/kiry163/easyllm"
	"github.com/kiry163/gopdfxt/internal/document"
)

func TestEngineRunnerRunAnalysisUsesToolCall(t *testing.T) {
	client := &toolClient{
		toolName: "submit_page_analysis",
		args: map[string]any{
			"page_type":        "body",
			"ignore_block_ids": []any{"p000-b001", "p000-b001"},
			"groups": []any{
				map[string]any{
					"kind":      "heading",
					"level":     1,
					"block_ids": []any{"p000-b000"},
				},
			},
		},
	}
	runner := NewEngineRunner(client, easyllm.Hooks{}, easyllm.ImageDetailLow)

	result, err := runner.RunAnalysis(context.Background(), document.Page{
		PageIndex: 0,
		Blocks: []document.Block{
			{BlockID: "p000-b000"},
			{BlockID: "p000-b001"},
		},
	}, "analysis prompt")
	if err != nil {
		t.Fatalf("RunAnalysis returned error: %v", err)
	}
	if result.PageType != "body" {
		t.Fatalf("expected body page, got %+v", result)
	}
	if len(result.IgnoreBlockIDs) != 1 || result.IgnoreBlockIDs[0] != "p000-b001" {
		t.Fatalf("expected deduped ignore ids, got %+v", result.IgnoreBlockIDs)
	}
	if len(result.Groups) != 1 || result.Groups[0].Kind != "heading" {
		t.Fatalf("unexpected analysis result: %+v", result)
	}
	if result.ModelCalls != 1 || result.Retries != 0 {
		t.Fatalf("unexpected stats: calls=%d retries=%d", result.ModelCalls, result.Retries)
	}
}

func TestEngineRunnerUsesConfiguredImageDetail(t *testing.T) {
	client := &toolClient{
		toolName: "submit_page_analysis",
		args: map[string]any{
			"page_type":        "non_body",
			"ignore_block_ids": []any{},
			"groups":           []any{},
		},
	}
	runner := NewEngineRunner(client, easyllm.Hooks{}, easyllm.ImageDetailLow)

	_, err := runner.RunAnalysis(context.Background(), document.Page{
		PageIndex:   0,
		ImageBase64: "aGVsbG8=",
	}, "analysis prompt")
	if err != nil {
		t.Fatalf("RunAnalysis returned error: %v", err)
	}
	if len(client.requests) != 1 {
		t.Fatalf("expected one model request, got %d", len(client.requests))
	}
	user, ok := client.requests[0].Input[0].(easyllm.UserMessageItem)
	if !ok {
		t.Fatalf("expected user message input, got %T", client.requests[0].Input[0])
	}
	if len(user.Content) == 0 || user.Content[0].Detail != easyllm.ImageDetailLow {
		t.Fatalf("expected low image detail, got %+v", user.Content)
	}
}

type toolClient struct {
	toolName string
	args     map[string]any
	requests []easyllm.ModelRequest
}

func (c *toolClient) Generate(ctx context.Context, req easyllm.ModelRequest) (*easyllm.ModelResponse, error) {
	c.requests = append(c.requests, req)
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

func (c *toolClient) GenerateStream(ctx context.Context, req easyllm.ModelRequest, handler easyllm.StreamHandler) error {
	panic("not implemented")
}
