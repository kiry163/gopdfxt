package render

import (
	"path/filepath"
	"strings"

	"github.com/kiry163/gopdfxt/internal/document"
)

type Article struct {
	Title   string
	Content string
	Start   int
	End     int
}

func SplitArticles(doc *document.Document, groups []document.Group) []Article {
	if len(groups) == 0 {
		return []Article{{
			Title:   fallbackArticleTitle(doc),
			Start:   defaultStartIndex(doc),
			End:     defaultEndIndex(doc),
			Content: "",
		}}
	}

	blockMap := make(map[string]document.Block)
	for _, page := range doc.Pages {
		for _, block := range page.Blocks {
			blockMap[block.BlockID] = block
		}
	}

	var headingIndexes []int
	for index, group := range groups {
		if group.Kind == "heading" && group.Level == 1 {
			headingIndexes = append(headingIndexes, index)
		}
	}

	if len(headingIndexes) == 0 {
		return []Article{buildArticle(doc, blockMap, fallbackArticleTitle(doc), groups)}
	}

	articles := make([]Article, 0, len(headingIndexes))
	for i, headingIndex := range headingIndexes {
		start := headingIndex
		if i == 0 {
			start = 0
		}
		end := len(groups)
		if i+1 < len(headingIndexes) {
			end = headingIndexes[i+1]
		}

		title := strings.TrimSpace(joinGroupText(blockMap, groups[headingIndex].BlockIDs))
		if title == "" {
			title = fallbackArticleTitle(doc)
		}

		articles = append(articles, buildArticle(doc, blockMap, title, groups[start:end]))
	}

	return articles
}

func buildArticle(doc *document.Document, blockMap map[string]document.Block, title string, groups []document.Group) Article {
	startIndex, endIndex, ok := articlePageRange(groups, blockMap)
	if !ok {
		startIndex = defaultStartIndex(doc)
		endIndex = defaultEndIndex(doc)
	}

	return Article{
		Title:   title,
		Start:   startIndex,
		End:     endIndex,
		Content: RenderContent(doc, groups),
	}
}

func articlePageRange(groups []document.Group, blockMap map[string]document.Block) (int, int, bool) {
	minPage := 0
	maxPage := 0
	found := false

	for _, group := range groups {
		for _, blockID := range group.BlockIDs {
			block, ok := blockMap[blockID]
			if !ok {
				continue
			}
			if !found {
				minPage = block.PageIndex
				maxPage = block.PageIndex
				found = true
				continue
			}
			if block.PageIndex < minPage {
				minPage = block.PageIndex
			}
			if block.PageIndex > maxPage {
				maxPage = block.PageIndex
			}
		}
	}

	return minPage, maxPage, found
}

func fallbackArticleTitle(doc *document.Document) string {
	if doc != nil {
		baseName := strings.TrimSpace(strings.TrimSuffix(filepath.Base(doc.PDF), filepath.Ext(doc.PDF)))
		if baseName != "" {
			return baseName
		}
	}
	return "article"
}

func defaultStartIndex(doc *document.Document) int {
	if doc == nil || doc.PageCount == 0 {
		return 0
	}
	return 0
}

func defaultEndIndex(doc *document.Document) int {
	if doc == nil || doc.PageCount == 0 {
		return 0
	}
	return doc.PageCount - 1
}
