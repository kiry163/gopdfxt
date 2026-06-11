package llm

import (
	"fmt"
	"sort"
	"strings"

	"github.com/kiry163/gopdfxt/internal/document"
)

type PageFilterResult struct {
	PageType       string
	IgnoreBlockIDs []string
}

type StructureResult struct {
	Groups []document.Group `json:"groups"`
}

type PageAnalysisResult struct {
	PageType       string
	IgnoreBlockIDs []string
	Groups         []document.Group
	ModelCalls     int
	Retries        int
}

func ValidatePageFilterResult(page document.Page, result *PageFilterResult) []string {
	if result == nil || result.PageType != "body" {
		return nil
	}

	if len(page.Blocks) == 0 {
		return []string{"body 页面没有可用 block"}
	}

	filteredPage := FilterPageBlocks(page, result)
	if len(filteredPage.Blocks) == 0 {
		return []string{"body 页面过滤后没有任何保留 block；如果整页都不应保留，请输出 non_body"}
	}
	return nil
}

func ValidateStructureResult(page document.Page, result *StructureResult) []string {
	var issues []string

	validIDs := make(map[string]struct{}, len(page.Blocks))
	for _, block := range page.Blocks {
		validIDs[block.BlockID] = struct{}{}
	}

	seen := make(map[string]string, len(page.Blocks))
	unknownIDs := map[string]struct{}{}
	duplicateIDs := map[string]struct{}{}

	recordID := func(blockID, source string) {
		if _, ok := validIDs[blockID]; !ok {
			unknownIDs[blockID] = struct{}{}
			return
		}
		if _, ok := seen[blockID]; ok {
			duplicateIDs[blockID] = struct{}{}
			return
		}
		seen[blockID] = source
	}

	for groupIndex, group := range result.Groups {
		for _, blockID := range group.BlockIDs {
			recordID(blockID, fmt.Sprintf("groups[%d]", groupIndex))
		}
	}

	if len(unknownIDs) > 0 {
		issues = append(issues, "包含未知 block_id: "+strings.Join(sortedSetValues(unknownIDs), ", "))
	}
	if len(duplicateIDs) > 0 {
		issues = append(issues, "同一 block_id 重复出现在 groups 中: "+strings.Join(sortedSetValues(duplicateIDs), ", "))
	}
	return issues
}

func FilterPageBlocks(page document.Page, filterResult *PageFilterResult) document.Page {
	dropSet := make(map[string]struct{}, len(filterResult.IgnoreBlockIDs))
	for _, blockID := range filterResult.IgnoreBlockIDs {
		dropSet[blockID] = struct{}{}
	}

	filtered := page
	filtered.Blocks = make([]document.Block, 0, len(page.Blocks))
	for _, block := range page.Blocks {
		if _, ok := dropSet[block.BlockID]; ok {
			continue
		}
		filtered.Blocks = append(filtered.Blocks, block)
	}
	return filtered
}

func RepairAnalysisResult(page document.Page, result *PageAnalysisResult) *PageAnalysisResult {
	if result == nil {
		result = &PageAnalysisResult{}
	}

	repaired := &PageAnalysisResult{
		PageType:   strings.TrimSpace(strings.ToLower(result.PageType)),
		ModelCalls: result.ModelCalls,
		Retries:    result.Retries,
	}
	if repaired.PageType != "non_body" {
		repaired.PageType = "body"
	}
	if repaired.PageType == "non_body" {
		return repaired
	}

	validIDs := make(map[string]struct{}, len(page.Blocks))
	for _, block := range page.Blocks {
		validIDs[block.BlockID] = struct{}{}
	}

	seen := make(map[string]struct{}, len(page.Blocks))
	for _, group := range result.Groups {
		group = normalizeGroup(group)
		var blockIDs []string
		for _, blockID := range group.BlockIDs {
			blockID = strings.TrimSpace(blockID)
			if blockID == "" {
				continue
			}
			if _, ok := validIDs[blockID]; !ok {
				continue
			}
			if _, ok := seen[blockID]; ok {
				continue
			}
			seen[blockID] = struct{}{}
			blockIDs = append(blockIDs, blockID)
		}
		if len(blockIDs) == 0 {
			continue
		}
		group.BlockIDs = blockIDs
		repaired.Groups = append(repaired.Groups, group)
	}

	if len(seen) == 0 {
		repaired.PageType = "non_body"
		return repaired
	}

	for _, block := range page.Blocks {
		if _, ok := seen[block.BlockID]; ok {
			continue
		}
		repaired.IgnoreBlockIDs = append(repaired.IgnoreBlockIDs, block.BlockID)
	}
	return repaired
}

func normalizeGroup(group document.Group) document.Group {
	kind := strings.TrimSpace(strings.ToLower(group.Kind))
	switch kind {
	case "heading":
		if group.Level != 1 {
			kind = "paragraph"
			group.Level = 0
		}
	case "paragraph", "formula", "image", "table":
		group.Level = 0
	default:
		kind = "paragraph"
		group.Level = 0
	}
	if kind != "heading" {
		group.Level = 0
	}
	group.Kind = kind
	return group
}

func sortedSetValues(values map[string]struct{}) []string {
	items := make([]string, 0, len(values))
	for value := range values {
		items = append(items, value)
	}
	sort.Strings(items)
	return items
}
