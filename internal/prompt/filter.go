package prompt

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/kiry163/gopdfxt/internal/document"
)

func BuildContentFilter(page document.Page) string {
	blockJSON := buildPageBlockJSON(page)

	return strings.TrimSpace(`
你正在处理一页学术文章或文章型 PDF。

参考图片与输入的文本块信息完成任务。

请判断：
1. 这一页是否整体不属于最终正文内容。
2. 如果这一页属于正文页，请找出其中明确应剔除的噪声 block。

应剔除的内容通常包括：
- 页眉
- 页脚
- 页码
- 重复刊名
- 卷期号
- 栏目名
- 纯导航性版式信息
- 目录页、目录续页、索引页中的目录条目

标题、作者、单位、摘要、关键词、正文、图表、公式、参考文献等文章内容应保留。

如果页面主要是在列出多篇不同文章的标题、作者、页码，或呈现大量点线连接的目录条目，这一页应判为 non_body。

如果拿不准某个 block 是否应剔除，不要输出它。
如果拿不准整页是否属于 non_body，优先判为 body。

请调用工具 submit_page_filter 提交判断结果，不要输出普通文本。

正文页工具参数：
- page_type: "body"
- ignore_block_ids: ["..."]

非正文页工具参数：
- page_type: "non_body"
- ignore_block_ids: []

要求：
- page_type 只能是 "body" 或 "non_body"
- 如果 page_type 是 "body"，必须输出 ignore_block_ids，允许为空数组
- ignore_block_ids 中只能出现当前页面已有的 block_id
- ignore_block_ids 不要重复

当前页面信息如下：
- page_index: ` + fmt.Sprintf("%d", page.PageIndex) + `
- page_width: ` + fmt.Sprintf("%.2f", page.Width) + `
- page_height: ` + fmt.Sprintf("%.2f", page.Height) + `
- total_blocks: ` + fmt.Sprintf("%d", len(page.Blocks)) + `

当前页面的 block 列表如下：
` + "\n" + string(blockJSON))
}

func BuildContentFilterRetry(page document.Page, previousOutput string, issues []string) string {
	blockJSON := buildPageBlockJSON(page)
	issueText := strings.Join(issues, "\n- ")
	if issueText != "" {
		issueText = "- " + issueText
	}

	return strings.TrimSpace(`
你刚才输出的 JSON 不合格，请修正后重新输出。
参考图片与输入的文本块信息完成任务。

上一版输出存在这些问题：
` + issueText + `

请重新调用工具 submit_page_filter，并严格满足：
- page_type 只能是 "body" 或 "non_body"
- 如果 page_type 是 "body"，必须输出 ignore_block_ids，允许为空数组
- ignore_block_ids 中只能包含当前页面已有的 block_id
- ignore_block_ids 不要重复
- 只有明确属于版式噪声的 block 才能放入 ignore_block_ids
- 目录页、目录续页、索引页中的目录条目属于版式噪声；如果整页主要是这类内容，应输出 "non_body"
- 如果页面主要是在列出多篇不同文章的标题、作者、页码，或呈现大量点线连接的目录条目，应输出 "non_body"
- 如果拿不准某个 block 是否应剔除，不要将它放入 ignore_block_ids
- 如果 page_type 是 "body"，ignore_block_ids 不能导致当前页没有任何保留 block；如果整页都不应保留，请输出 "non_body"
- 如果拿不准整页是否属于 non_body，优先判为 body
- 只调用工具，不要解释，不要代码块

你上一版输出如下：
` + "\n" + previousOutput + `

当前页面信息如下：
- page_index: ` + fmt.Sprintf("%d", page.PageIndex) + `
- page_width: ` + fmt.Sprintf("%.2f", page.Width) + `
- page_height: ` + fmt.Sprintf("%.2f", page.Height) + `
- total_blocks: ` + fmt.Sprintf("%d", len(page.Blocks)) + `

当前页面的 block 列表如下：
` + "\n" + string(blockJSON))
}

func buildPageBlockJSON(page document.Page) []byte {
	type blockView struct {
		BlockID   string    `json:"block_id"`
		BlockType string    `json:"block_type"`
		Text      string    `json:"text"`
		FontSize  float64   `json:"font_size"`
		LineCount int       `json:"line_count"`
		BBox      []float64 `json:"bbox"`
	}

	views := make([]blockView, 0, len(page.Blocks))
	for _, block := range page.Blocks {
		views = append(views, blockView{
			BlockID:   block.BlockID,
			BlockType: block.BlockType,
			Text:      block.Text,
			FontSize:  block.FontSize,
			LineCount: block.LineCount,
			BBox:      block.BBox,
		})
	}
	blockJSON, _ := json.MarshalIndent(views, "", "  ")
	return blockJSON
}
