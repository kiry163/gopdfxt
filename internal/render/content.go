package render

import (
	"regexp"
	"strings"
	"unicode"

	"github.com/kiry163/gopdfxt/internal/document"
)

var numberedHeadingPattern = regexp.MustCompile(`^(?:[一二三四五六七八九十百千]+[、.．]|[（(][一二三四五六七八九十百千]+[）)]|[0-9]+(?:[.．、]|\s+))`)

func BuildGroups(doc *document.Document, pageResults []document.PageStructure) []document.Group {
	var groups []document.Group
	for i, page := range doc.Pages {
		if i >= len(pageResults) {
			break
		}
		groups = append(groups, buildRenderableGroups(page, pageResults[i])...)
	}

	groups = mergeParagraphContinuations(doc, groups)
	return normalizeHeadings(doc, groups)
}

func RenderContent(doc *document.Document, groups []document.Group) string {
	blockMap := map[string]document.Block{}
	for _, page := range doc.Pages {
		for _, block := range page.Blocks {
			blockMap[block.BlockID] = block
		}
	}

	var output []string
	for _, group := range groups {
		if group.Kind == "heading" {
			continue
		}
		text := joinGroupText(blockMap, group.BlockIDs)
		if group.Kind != "image" && strings.TrimSpace(text) == "" {
			continue
		}
		switch group.Kind {
		case "formula":
			output = append(output, "[公式]")
		case "image":
			output = append(output, "[图片]")
		case "table":
			output = append(output, "[表格]")
		default:
			output = append(output, strings.TrimSpace(text))
		}
	}

	if len(output) == 0 {
		return ""
	}
	return strings.Join(output, "\n\n") + "\n"
}

func buildRenderableGroups(page document.Page, pageResult document.PageStructure) []document.Group {
	if pageResult.PageType == "non_body" {
		return nil
	}

	ignored := map[string]struct{}{}
	for _, blockID := range pageResult.IgnoreBlockIDs {
		ignored[blockID] = struct{}{}
	}

	var groups []document.Group
	for _, group := range pageResult.Groups {
		var kept []string
		seen := map[string]struct{}{}
		for _, blockID := range group.BlockIDs {
			if _, skip := ignored[blockID]; skip {
				continue
			}
			if _, dup := seen[blockID]; dup {
				continue
			}
			seen[blockID] = struct{}{}
			kept = append(kept, blockID)
		}
		if len(kept) == 0 {
			continue
		}

		group.BlockIDs = kept
		switch group.Kind {
		case "heading":
			if group.Level != 1 {
				group.Kind = "paragraph"
				group.Level = 0
			}
		case "paragraph", "formula", "image", "table":
			group.Level = 0
		default:
			group.Kind = "paragraph"
			group.Level = 0
		}
		groups = append(groups, group)
	}
	return groups
}

func mergeParagraphContinuations(doc *document.Document, groups []document.Group) []document.Group {
	if len(groups) < 2 {
		return groups
	}

	blockMap := map[string]document.Block{}
	pageMap := map[int]document.Page{}
	for _, page := range doc.Pages {
		pageMap[page.PageIndex] = page
		for _, block := range page.Blocks {
			blockMap[block.BlockID] = block
		}
	}

	merged := make([]document.Group, 0, len(groups))
	current := groups[0]
	for i := 1; i < len(groups); i++ {
		next := groups[i]
		if shouldMergeParagraphGroups(current, next, blockMap, pageMap) {
			current.BlockIDs = append(current.BlockIDs, next.BlockIDs...)
			continue
		}
		merged = append(merged, current)
		current = next
	}
	merged = append(merged, current)
	return merged
}

func shouldMergeParagraphGroups(left, right document.Group, blockMap map[string]document.Block, pageMap map[int]document.Page) bool {
	if left.Kind != "paragraph" || right.Kind != "paragraph" {
		return false
	}
	if len(left.BlockIDs) == 0 || len(right.BlockIDs) == 0 {
		return false
	}

	leftBlock, ok := blockMap[left.BlockIDs[len(left.BlockIDs)-1]]
	if !ok {
		return false
	}
	rightBlock, ok := blockMap[right.BlockIDs[0]]
	if !ok {
		return false
	}

	leftPage, ok := pageMap[leftBlock.PageIndex]
	if !ok {
		return false
	}
	rightPage, ok := pageMap[rightBlock.PageIndex]
	if !ok {
		return false
	}

	pageGap := rightBlock.PageIndex - leftBlock.PageIndex
	if pageGap < 0 || pageGap > 1 {
		return false
	}
	if absFloat(leftBlock.FontSize-rightBlock.FontSize) > 1.0 {
		return false
	}

	leftText := normalizeBlockText(leftBlock.Text)
	rightText := normalizeBlockText(rightBlock.Text)
	if leftText == "" || rightText == "" {
		return false
	}
	if endsWithParagraphTerminal(leftText) {
		return false
	}
	if looksLikeNewStructuralStart(rightText) {
		return false
	}

	if pageGap == 1 {
		return isNearBottom(leftBlock, leftPage) && isNearTop(rightBlock, rightPage)
	}
	if !isNearBottom(leftBlock, leftPage) || !isNearTop(rightBlock, rightPage) {
		return false
	}
	if rightBlock.BBox[1] >= leftBlock.BBox[1] {
		return false
	}
	if !isLikelyColumnJump(leftBlock, rightBlock, leftPage) {
		return false
	}
	return true
}

func normalizeHeadings(doc *document.Document, groups []document.Group) []document.Group {
	blockMap := map[string]document.Block{}
	for _, page := range doc.Pages {
		for _, block := range page.Blocks {
			blockMap[block.BlockID] = block
		}
	}

	normalized := make([]document.Group, 0, len(groups))
	for _, group := range groups {
		if group.Kind == "heading" && group.Level == 1 {
			text := strings.TrimSpace(joinGroupText(blockMap, group.BlockIDs))
			if text == "" || numberedHeadingPattern.MatchString(text) {
				group.Kind = "paragraph"
				group.Level = 0
			}
		}
		normalized = append(normalized, group)
	}
	return normalized
}

func joinGroupText(blockMap map[string]document.Block, blockIDs []string) string {
	text := ""
	for _, blockID := range blockIDs {
		block, ok := blockMap[blockID]
		if !ok {
			continue
		}
		text = appendText(text, normalizeBlockText(block.Text))
	}
	return text
}

func normalizeBlockText(text string) string {
	lines := strings.Split(text, "\n")
	parts := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts = append(parts, line)
	}

	joined := ""
	for _, part := range parts {
		joined = appendText(joined, part)
	}
	return joined
}

func appendText(left, right string) string {
	if left == "" {
		return right
	}
	if needsSpace(left, right) {
		return left + " " + right
	}
	return left + right
}

func needsSpace(left, right string) bool {
	if left == "" || right == "" {
		return false
	}
	leftRunes := []rune(left)
	rightRunes := []rune(right)
	if unicode.IsSpace(leftRunes[len(leftRunes)-1]) || unicode.IsSpace(rightRunes[0]) {
		return false
	}
	if unicode.Is(unicode.Han, leftRunes[len(leftRunes)-1]) || unicode.Is(unicode.Han, rightRunes[0]) {
		return false
	}
	if strings.ContainsRune("，。！？；：、)]】》", rightRunes[0]) {
		return false
	}
	if strings.ContainsRune("([【《", leftRunes[len(leftRunes)-1]) {
		return false
	}
	return true
}

func endsWithParagraphTerminal(text string) bool {
	text = strings.TrimSpace(text)
	if text == "" {
		return false
	}

	runes := []rune(text)
	for len(runes) > 0 {
		last := runes[len(runes)-1]
		if unicode.IsSpace(last) {
			runes = runes[:len(runes)-1]
			continue
		}
		if strings.ContainsRune("”’\"')）】》]", last) {
			runes = runes[:len(runes)-1]
			continue
		}
		return strings.ContainsRune("。！？；：.!?;:", last)
	}
	return false
}

func looksLikeNewStructuralStart(text string) bool {
	text = strings.TrimSpace(text)
	if text == "" {
		return false
	}
	if strings.HasPrefix(text, "#") || strings.HasPrefix(text, "##") {
		return true
	}
	if strings.HasPrefix(text, "[") && len(text) > 1 {
		next := []rune(text)[1]
		if unicode.IsDigit(next) {
			return true
		}
	}
	runes := []rune(text)
	if len(runes) >= 2 && unicode.IsDigit(runes[0]) && strings.ContainsRune(".．、", runes[1]) {
		return true
	}
	return false
}

func isNearBottom(block document.Block, page document.Page) bool {
	if len(block.BBox) < 4 || page.Height <= 0 {
		return false
	}
	return block.BBox[3] >= page.Height*0.82
}

func isNearTop(block document.Block, page document.Page) bool {
	if len(block.BBox) < 4 || page.Height <= 0 {
		return false
	}
	return block.BBox[1] <= page.Height*0.18
}

func isLikelyColumnJump(left, right document.Block, page document.Page) bool {
	if len(left.BBox) < 4 || len(right.BBox) < 4 || page.Width <= 0 {
		return false
	}
	return absFloat(right.BBox[0]-left.BBox[0]) >= page.Width*0.18
}

func absFloat(value float64) float64 {
	if value < 0 {
		return -value
	}
	return value
}
