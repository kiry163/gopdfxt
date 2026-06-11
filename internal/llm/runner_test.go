package llm

import (
	"context"
	"strings"
	"testing"

	"github.com/kiry163/easyllm"
	"github.com/kiry163/gopdfxt/internal/document"
)

func TestEngineRunnerRunAnalysisUsesToolCall(t *testing.T) {
	client := &toolClient{
		toolName: "submit_page_analysis",
		args: map[string]any{
			"page_type": "body",
			"groups": []any{
				map[string]any{
					"kind":  "heading",
					"level": 1,
					"ranges": []any{
						map[string]any{
							"start_block_id": "p000-b000",
							"end_block_id":   "p000-b000",
						},
					},
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
		t.Fatalf("expected omitted block to be ignored, got %+v", result.IgnoreBlockIDs)
	}
	if len(result.Groups) != 1 || result.Groups[0].Kind != "heading" || len(result.Groups[0].BlockIDs) != 1 || result.Groups[0].BlockIDs[0] != "p000-b000" {
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
			"page_type": "non_body",
			"groups":    []any{},
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

func TestEngineRunnerExpandsMultipleRangesInOneGroup(t *testing.T) {
	client := &toolClient{
		toolName: "submit_page_analysis",
		args: map[string]any{
			"page_type": "body",
			"groups": []any{
				map[string]any{
					"kind": "paragraph",
					"ranges": []any{
						map[string]any{
							"start_block_id": "p000-b000",
							"end_block_id":   "p000-b001",
						},
						map[string]any{
							"start_block_id": "p000-b003",
							"end_block_id":   "p000-b003",
						},
					},
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
			{BlockID: "p000-b002"},
			{BlockID: "p000-b003"},
		},
	}, "analysis prompt")
	if err != nil {
		t.Fatalf("RunAnalysis returned error: %v", err)
	}
	if len(result.Groups) != 1 {
		t.Fatalf("expected one group, got %+v", result.Groups)
	}
	want := []string{"p000-b000", "p000-b001", "p000-b003"}
	if strings.Join(result.Groups[0].BlockIDs, ",") != strings.Join(want, ",") {
		t.Fatalf("expected ranges to expand to %v, got %v", want, result.Groups[0].BlockIDs)
	}
	if len(result.IgnoreBlockIDs) != 1 || result.IgnoreBlockIDs[0] != "p000-b002" {
		t.Fatalf("expected omitted middle block to be ignored, got %+v", result.IgnoreBlockIDs)
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
