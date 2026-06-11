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
	PageType string      `tool:"name=page_type,required,desc=body for content pages, non_body for table of contents/index/navigation pages,enum=body|non_body"`
	Groups   []groupArgs `tool:"name=groups,desc=Reading-order groups for retained content on body pages; omitted blocks are ignored"`
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
	return "Submit page content classification and retained block reading-order groups."
}

func (t pageAnalysisTool) Run(ctx context.Context, call easyllm.ToolCallContext, args pageAnalysisArgs) (easyllm.ToolResult, error) {
	result := &PageAnalysisResult{
		PageType: strings.TrimSpace(strings.ToLower(args.PageType)),
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
		blockIDs, err := expandRanges(t.page, group.Ranges)
		if err != nil {
			return easyllm.ToolResult{}, validationError([]string{err.Error()})
		}
		result.Groups = append(result.Groups, document.Group{
			Kind:     kind,
			Level:    level,
			BlockIDs: blockIDs,
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
			"page_type": result.PageType,
			"groups":    result.Groups,
		},
	}, nil
}

type groupArgs struct {
	Kind   string      `tool:"name=kind,required,enum=heading|paragraph|formula|image|table"`
	Level  int         `tool:"name=level,desc=Use 1 only for whole-article title headings"`
	Ranges []rangeArgs `tool:"name=ranges,required,minItems=1,desc=Retained block ranges in reading order"`
}

type rangeArgs struct {
	StartBlockID string `tool:"name=start_block_id,required,desc=First block_id in this retained range"`
	EndBlockID   string `tool:"name=end_block_id,required,desc=Last block_id in this retained range"`
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

func expandRanges(page document.Page, ranges []rangeArgs) ([]string, error) {
	indexByID := make(map[string]int, len(page.Blocks))
	for index, block := range page.Blocks {
		indexByID[block.BlockID] = index
	}

	var blockIDs []string
	for rangeIndex, item := range ranges {
		startID := strings.TrimSpace(item.StartBlockID)
		endID := strings.TrimSpace(item.EndBlockID)
		if startID == "" || endID == "" {
			return nil, fmt.Errorf("groups range %d must include start_block_id and end_block_id", rangeIndex)
		}
		start, ok := indexByID[startID]
		if !ok {
			return nil, fmt.Errorf("groups range %d contains unknown start_block_id: %s", rangeIndex, startID)
		}
		end, ok := indexByID[endID]
		if !ok {
			return nil, fmt.Errorf("groups range %d contains unknown end_block_id: %s", rangeIndex, endID)
		}
		if start > end {
			return nil, fmt.Errorf("groups range %d has start_block_id after end_block_id: %s > %s", rangeIndex, startID, endID)
		}
		for i := start; i <= end; i++ {
			blockIDs = append(blockIDs, page.Blocks[i].BlockID)
		}
	}
	return blockIDs, nil
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
