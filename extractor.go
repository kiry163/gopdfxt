package gopdfxt

import "context"

type Extractor interface {
	Extract(ctx context.Context, input PDFInput) (*ExtractedDocument, error)
}

type PDFInput struct {
	Path string
}

type ExtractedDocument struct {
	PDF       string
	PageCount int
	Pages     []ExtractedPage
}

type ExtractedPage struct {
	PageIndex   int
	Width       float64
	Height      float64
	ImageBase64 string
	Blocks      []ExtractedBlock
}

type ExtractedBlock struct {
	BlockID   string
	PageIndex int
	BlockType string
	Text      string
	BBox      []float64
	FontSize  float64
	Fonts     []string
	LineCount int
}
