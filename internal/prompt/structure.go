package prompt

import (
	"fmt"
	"strings"

	"github.com/kiry163/gopdfxt/internal/document"
)

func BuildStructure(page document.Page) string {
	blockJSON := buildPageBlockJSON(page)

	return strings.TrimSpace(`
你正在处理一页学术文章或文章型 PDF。
参考图片与输入的文本块信息完成任务。

请将当前页面中这些已保留的 block，按最终阅读顺序组织为 groups。

请调用工具 submit_page_structure 提交 groups，不要输出普通文本、解释或 Markdown 代码块。

工具参数包含 groups 数组；每个 group 需要 kind、block_ids，主标题 heading 还需要 level=1。

硬性要求：
1. 当前页给出的每个 block_id 必须且只能出现一次，并且必须出现在 groups 中。
2. 不能遗漏任何 block_id。
3. 如果拿不准某个 block 的类型，优先放入 groups，并归为 paragraph。

结构要求：
1. groups 必须是最终阅读顺序。
2. 对双栏页面，通常先读完整个左栏，再读右栏。
3. 如果页面顶部出现的是上一页正文的续段，应先输出续段，再输出该页后面才开始的新标题。
4. 同一段正文如果只是因为换栏或分页而延续，不要拆错顺序；必要时应放在同一个 paragraph group 中。

标题要求：
1. heading 只允许用于整篇文章的主标题，也就是 level=1。
2. 除主标题外，其他任何标题、节标题、小节标题、带编号标题，一律不要标成 heading，而应归为 paragraph。
3. 对 level=1 要非常保守；拿不准时，归为 paragraph。

图片、公式、表格要求：
1. 独立图片块应使用 kind="image"。
2. 独立展示公式应使用 kind="formula"。
3. 独立表格或表格区域应使用 kind="table"。
4. 如果拿不准是否独立，优先归入 paragraph，而不是省略。

当前页面信息如下：
- page_index: ` + fmt.Sprintf("%d", page.PageIndex) + `
- page_width: ` + fmt.Sprintf("%.2f", page.Width) + `
- page_height: ` + fmt.Sprintf("%.2f", page.Height) + `
- total_blocks: ` + fmt.Sprintf("%d", len(page.Blocks)) + `

当前页面的已保留 block 列表如下：
` + "\n" + string(blockJSON))
}

func BuildStructureRetry(page document.Page, previousOutput string, issues []string) string {
	blockJSON := buildPageBlockJSON(page)
	issueText := strings.Join(issues, "\n- ")
	if issueText != "" {
		issueText = "- " + issueText
	}

	return strings.TrimSpace(`
你刚才输出的第二阶段 JSON 不合格，需要严格修正。
参考图片与输入的文本块信息完成任务。

上一版输出存在这些问题：
` + issueText + `

请重新调用工具 submit_page_structure，并严格满足：
- 每个 block_id 必须且只能出现一次，并且必须出现在 groups 中
- 必须覆盖当前页给出的全部 block_id，不能遗漏
- groups 必须是最终阅读顺序
- 如果拿不准某个 block 的类型，优先归入 paragraph，不要省略
- 只调用工具，不要解释，不要代码块

你上一版输出如下：
` + "\n" + previousOutput + `

当前页面信息如下：
- page_index: ` + fmt.Sprintf("%d", page.PageIndex) + `
- page_width: ` + fmt.Sprintf("%.2f", page.Width) + `
- page_height: ` + fmt.Sprintf("%.2f", page.Height) + `
- total_blocks: ` + fmt.Sprintf("%d", len(page.Blocks)) + `

当前页面的已保留 block 列表如下：
` + "\n" + string(blockJSON))
}
