package llm

import (
	"context"
	"fmt"
	"strings"

	"github.com/kiry163/easyllm"
	"github.com/kiry163/gopdfxt/internal/document"
)

type EngineRunner struct {
	client easyllm.Client
	hooks  easyllm.Hooks
}

func NewEngineRunner(client easyllm.Client, hooks easyllm.Hooks) *EngineRunner {
	return &EngineRunner{client: client, hooks: hooks}
}

func (r *EngineRunner) RunFilter(ctx context.Context, page document.Page, prompt string) (*PageFilterResult, error) {
	store := &pageFilterStore{}
	tool, err := easyllm.NewTool[pageFilterArgs](pageFilterTool{page: page, store: store}, easyllm.WithStrict(true))
	if err != nil {
		return nil, err
	}
	if err := r.runTool(ctx, page, prompt, tool); err != nil {
		return nil, err
	}
	if store.result == nil {
		return nil, fmt.Errorf("page filter tool did not submit a result")
	}
	return store.result, nil
}

func (r *EngineRunner) RunStructure(ctx context.Context, page document.Page, prompt string) (*StructureResult, error) {
	store := &structureStore{}
	tool, err := easyllm.NewTool[pageStructureArgs](pageStructureTool{page: page, store: store}, easyllm.WithStrict(true))
	if err != nil {
		return nil, err
	}
	if err := r.runTool(ctx, page, prompt, tool); err != nil {
		return nil, err
	}
	if store.result == nil {
		return nil, fmt.Errorf("page structure tool did not submit a result")
	}
	return store.result, nil
}

func (r *EngineRunner) runTool(ctx context.Context, page document.Page, prompt string, tool easyllm.Tool) error {
	engine := easyllm.NewEngine(
		r.client,
		easyllm.WithTools(tool),
		easyllm.WithHooks(r.hooks),
		easyllm.WithStopAfterToolCall(true),
		easyllm.WithMaxModelCalls(3),
	)
	_, err := engine.Run(ctx, easyllm.RunRequest{
		InputParts: []easyllm.ContentPart{
			easyllm.NewImagePart("data:image/png;base64,"+page.ImageBase64, easyllm.ImageDetailAuto),
			easyllm.NewTextPart(prompt),
		},
		Metadata: map[string]any{
			"page_index": page.PageIndex,
			"tool":       tool.Definition().Name,
		},
	})
	return err
}

type pageFilterArgs struct {
	PageType       string   `tool:"name=page_type,required,desc=body for content pages, non_body for table of contents/index/navigation pages,enum=body|non_body"`
	IgnoreBlockIDs []string `tool:"name=ignore_block_ids,desc=Block IDs that are layout noise on body pages"`
}

type pageFilterStore struct {
	result *PageFilterResult
}

type pageFilterTool struct {
	page  document.Page
	store *pageFilterStore
}

func (pageFilterTool) Name() string {
	return "submit_page_filter"
}

func (pageFilterTool) Description() string {
	return "Submit whether the current PDF page is body content and which blocks should be ignored as layout noise."
}

func (t pageFilterTool) Run(ctx context.Context, call easyllm.ToolCallContext, args pageFilterArgs) (easyllm.ToolResult, error) {
	result := &PageFilterResult{
		PageType:       strings.TrimSpace(strings.ToLower(args.PageType)),
		IgnoreBlockIDs: uniqueStrings(args.IgnoreBlockIDs),
	}
	issues := ValidatePageFilterResult(t.page, result)
	if len(issues) > 0 {
		return easyllm.ToolResult{}, validationError(issues)
	}
	t.store.result = result
	return easyllm.ToolResult{
		Message: "page filter accepted",
		Data: map[string]any{
			"page_type":        result.PageType,
			"ignore_block_ids": result.IgnoreBlockIDs,
		},
	}, nil
}

type pageStructureArgs struct {
	Groups []groupArgs `tool:"name=groups,required,minItems=1,desc=Reading-order groups covering every retained block exactly once"`
}

type groupArgs struct {
	Kind     string   `tool:"name=kind,required,enum=heading|paragraph|formula|image|table"`
	Level    int      `tool:"name=level,desc=Use 1 only for whole-article title headings"`
	BlockIDs []string `tool:"name=block_ids,required,minItems=1"`
}

type structureStore struct {
	result *StructureResult
}

type pageStructureTool struct {
	page  document.Page
	store *structureStore
}

func (pageStructureTool) Name() string {
	return "submit_page_structure"
}

func (pageStructureTool) Description() string {
	return "Submit reading-order groups for the retained PDF page blocks."
}

func (t pageStructureTool) Run(ctx context.Context, call easyllm.ToolCallContext, args pageStructureArgs) (easyllm.ToolResult, error) {
	result := &StructureResult{
		Groups: make([]document.Group, 0, len(args.Groups)),
	}
	for _, group := range args.Groups {
		kind := strings.TrimSpace(strings.ToLower(group.Kind))
		level := group.Level
		if kind == "heading" && level != 1 {
			kind = "paragraph"
			level = 0
		}
		if kind != "heading" {
			level = 0
		}
		result.Groups = append(result.Groups, document.Group{
			Kind:     kind,
			Level:    level,
			BlockIDs: uniqueStrings(group.BlockIDs),
		})
	}
	issues := ValidateStructureResult(t.page, result)
	if len(issues) > 0 {
		return easyllm.ToolResult{}, validationError(issues)
	}
	t.store.result = result
	return easyllm.ToolResult{
		Message: "page structure accepted",
		Data: map[string]any{
			"groups": result.Groups,
		},
	}, nil
}

func validationError(issues []string) error {
	return easyllm.NewPayloadError(strings.Join(issues, "; "), map[string]any{
		"status": "error",
		"code":   "validation_failed",
		"issues": issues,
	})
}

func uniqueStrings(values []string) []string {
	seen := map[string]struct{}{}
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}
