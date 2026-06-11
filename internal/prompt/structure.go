package prompt

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/kiry163/gopdfxt/internal/document"
)

func BuildAnalysis(page document.Page) string {
	blockJSON := buildPageBlockJSON(page)

	return strings.TrimSpace(`
你正在处理一页学术文章或文章型 PDF。
参考图片与输入的文本块信息完成任务。

请一次性完成：
1. 判断当前页是否属于最终正文内容。
2. 如果是正文页，将应保留的正文内容按最终阅读顺序组织为 groups。

请调用工具 submit_page_analysis 提交结果，不要输出普通文本、解释或 Markdown 代码块。

工具参数：
- page_type: "body" 或 "non_body"
- groups: 正文页的阅读顺序分组；每个 group 使用 ranges 指定保留 block 范围；非正文页为空数组

非正文页判断：
- 如果页面主要是目录、索引、导航信息，或列出多篇文章标题/作者/页码，应判为 non_body。
- 如果拿不准整页是否属于 non_body，优先判为 body。

保留与忽略：
- 页眉、页脚、页码、重复刊名、卷期号、栏目名、纯导航性版式信息通常应剔除。
- 标题、作者、单位、摘要、关键词、正文、图表、公式、参考文献等文章内容应保留。
- 只输出应保留内容的 groups；未出现在 groups 中的 block 会被忽略。
- 如果拿不准某个 block 是否属于正文内容，优先保留。

groups 要求：
1. page_type 为 "body" 时，groups 必须按最终阅读顺序输出。
2. 每个 group 必须包含 ranges；每个 range 使用 start_block_id 和 end_block_id 表示当前 block 列表中的连续范围，首尾都包含。
3. 一个 group 可以包含多个 ranges，用于表达跨栏、跨区域或中间夹有噪声 block 的同一段内容。
4. 不要让同一个 block 出现在多个 group 或多个 range 中。
5. 对双栏页面，通常先读完整个左栏，再读右栏。
6. 如果页面顶部出现的是上一页正文续段，应先输出续段，再输出该页后面才开始的新标题。
7. 同一段正文如果只是因为换栏或分页而延续，必要时放在同一个 paragraph group 中。
8. 如果拿不准某个 block 的类型，归为 paragraph。

标题要求：
1. heading 只允许用于整篇文章的主标题，也就是 level=1。
2. 除主标题外，其他任何标题、节标题、小节标题、带编号标题，一律不要标成 heading，而应归为 paragraph。
3. 对 level=1 要非常保守；拿不准时归为 paragraph。

图片、公式、表格要求：
1. 独立图片块使用 kind="image"。
2. 独立展示公式使用 kind="formula"。
3. 独立表格或表格区域使用 kind="table"。
4. 如果拿不准是否独立，优先归入 paragraph。

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
