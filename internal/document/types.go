package document

type Document struct {
	PDF       string `json:"pdf"`
	PageCount int    `json:"page_count"`
	Pages     []Page `json:"pages"`
}

type Page struct {
	PageIndex   int     `json:"page_index"`
	Width       float64 `json:"width"`
	Height      float64 `json:"height"`
	ImageBase64 string  `json:"image_base64"`
	Blocks      []Block `json:"blocks"`
}

type Block struct {
	BlockID   string    `json:"block_id"`
	PageIndex int       `json:"page_index"`
	BlockType string    `json:"block_type"`
	Text      string    `json:"text"`
	BBox      []float64 `json:"bbox"`
	FontSize  float64   `json:"font_size"`
	Fonts     []string  `json:"fonts"`
	LineCount int       `json:"line_count"`
}

type Group struct {
	Kind     string   `json:"kind"`
	Level    int      `json:"level,omitempty"`
	BlockIDs []string `json:"block_ids"`
}

type PageStructure struct {
	PageType       string   `json:"page_type"`
	IgnoreBlockIDs []string `json:"ignore_block_ids"`
	Groups         []Group  `json:"groups"`
}
