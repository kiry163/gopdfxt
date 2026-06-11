package llm

import (
	"context"
	"fmt"
	"strings"

	"github.com/kiry163/easyllm"
	"github.com/kiry163/gopdfxt/internal/document"
)

type EngineRunner struct {
	client      easyllm.Client
	hooks       easyllm.Hooks
	imageDetail easyllm.ImageDetail
}

func NewEngineRunner(client easyllm.Client, hooks easyllm.Hooks, imageDetail easyllm.ImageDetail) *EngineRunner {
	if imageDetail == "" {
		imageDetail = easyllm.ImageDetailAuto
	}
	return &EngineRunner{client: client, hooks: hooks, imageDetail: imageDetail}
}

func (r *EngineRunner) RunAnalysis(ctx context.Context, page document.Page, prompt string) (*PageAnalysisResult, error) {
	store := &pageAnalysisStore{}
	tool, err := easyllm.NewTool[pageAnalysisArgs](pageAnalysisTool{page: page, store: store}, easyllm.WithStrict(true))
	if err != nil {
		return nil, err
	}
	stats, err := r.runTool(ctx, page, prompt, tool)
	if err != nil {
		return nil, err
	}
	if store.result == nil {
		return nil, fmt.Errorf("page analysis tool did not submit a result")
	}
	store.result.ModelCalls = stats.ModelCalls
	store.result.Retries = stats.Retries
	return store.result, nil
}

type runStats struct {
	ModelCalls int
	Retries    int
}

func (r *EngineRunner) runTool(ctx context.Context, page document.Page, prompt string, tool easyllm.Tool) (runStats, error) {
	var stats runStats
	engine := easyllm.NewEngine(
		r.client,
		easyllm.WithTools(tool),
		easyllm.WithHooks(r.countingHooks(&stats)),
		easyllm.WithStopAfterToolCall(true),
		easyllm.WithMaxModelCalls(3),
	)
	_, err := engine.Run(ctx, easyllm.RunRequest{
		InputParts: []easyllm.ContentPart{
			easyllm.NewImagePart("data:image/png;base64,"+page.ImageBase64, r.imageDetail),
			easyllm.NewTextPart(prompt),
		},
		Metadata: map[string]any{
			"page_index": page.PageIndex,
			"tool":       tool.Definition().Name,
		},
	})
	if stats.ModelCalls > 0 {
		stats.Retries = stats.ModelCalls - 1
	}
	return stats, err
}

func (r *EngineRunner) countingHooks(stats *runStats) easyllm.Hooks {
	hooks := r.hooks
	originalOnModelRequest := hooks.OnModelRequest
	hooks.OnModelRequest = func(e easyllm.ModelRequestEvent) error {
		stats.ModelCalls++
		if originalOnModelRequest != nil {
			return originalOnModelRequest(e)
		}
		return nil
	}
	return hooks
}

type pageAnalysisArgs struct {
	PageType       string      `tool:"name=page_type,required,desc=body for content pages, non_body for table of contents/index/navigation pages,enum=body|non_body"`
	IgnoreBlockIDs []string    `tool:"name=ignore_block_ids,desc=Block IDs that are layout noise on body pages"`
	Groups         []groupArgs `tool:"name=groups,desc=Reading-order groups covering every retained block exactly once on body pages"`
}

type pageAnalysisStore struct {
	result *PageAnalysisResult
}

type pageAnalysisTool struct {
	page  document.Page
	store *pageAnalysisStore
}

func (pageAnalysisTool) Name() string {
	return "submit_page_analysis"
}

func (pageAnalysisTool) Description() string {
	return "Submit page content classification, ignored layout-noise blocks, and retained block reading-order groups."
}

func (t pageAnalysisTool) Run(ctx context.Context, call easyllm.ToolCallContext, args pageAnalysisArgs) (easyllm.ToolResult, error) {
	result := &PageAnalysisResult{
		PageType:       strings.TrimSpace(strings.ToLower(args.PageType)),
		IgnoreBlockIDs: uniqueStrings(args.IgnoreBlockIDs),
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
	result = RepairAnalysisResult(t.page, result)
	issues := validateAnalysisResult(t.page, result)
	if len(issues) > 0 {
		return easyllm.ToolResult{}, validationError(issues)
	}
	t.store.result = result
	return easyllm.ToolResult{
		Message: "page analysis accepted",
		Data: map[string]any{
			"page_type":        result.PageType,
			"ignore_block_ids": result.IgnoreBlockIDs,
			"groups":           result.Groups,
		},
	}, nil
}

type groupArgs struct {
	Kind     string   `tool:"name=kind,required,enum=heading|paragraph|formula|image|table"`
	Level    int      `tool:"name=level,desc=Use 1 only for whole-article title headings"`
	BlockIDs []string `tool:"name=block_ids,required,minItems=1"`
}

func validateAnalysisResult(page document.Page, result *PageAnalysisResult) []string {
	if result == nil || result.PageType == "non_body" {
		return nil
	}
	filterResult := &PageFilterResult{
		PageType:       result.PageType,
		IgnoreBlockIDs: result.IgnoreBlockIDs,
	}
	if issues := ValidatePageFilterResult(page, filterResult); len(issues) > 0 {
		return issues
	}
	filtered := FilterPageBlocks(page, filterResult)
	return ValidateStructureResult(filtered, &StructureResult{Groups: result.Groups})
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
