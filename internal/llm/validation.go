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

	var missingIDs []string
	for _, block := range page.Blocks {
		if _, ok := seen[block.BlockID]; !ok {
			missingIDs = append(missingIDs, block.BlockID)
		}
	}

	if len(unknownIDs) > 0 {
		issues = append(issues, "包含未知 block_id: "+strings.Join(sortedSetValues(unknownIDs), ", "))
	}
	if len(duplicateIDs) > 0 {
		issues = append(issues, "同一 block_id 重复出现在 groups 中: "+strings.Join(sortedSetValues(duplicateIDs), ", "))
	}
	if len(missingIDs) > 0 {
		issues = append(issues, "遗漏了未处理的 block_id: "+strings.Join(missingIDs, ", "))
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

func sortedSetValues(values map[string]struct{}) []string {
	items := make([]string, 0, len(values))
	for value := range values {
		items = append(items, value)
	}
	sort.Strings(items)
	return items
}
