package gopdfxt

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/gen2brain/go-fitz"
)

type DefaultExtractor struct {
	DPI float64
}

func NewDefaultExtractor() *DefaultExtractor {
	return &DefaultExtractor{DPI: 144}
}

func (e *DefaultExtractor) Extract(ctx context.Context, input PDFInput) (*ExtractedDocument, error) {
	doc, err := fitz.New(input.Path)
	if err != nil {
		return nil, err
	}
	defer doc.Close()

	pageCount := doc.NumPage()
	result := &ExtractedDocument{
		PDF:       input.Path,
		PageCount: pageCount,
		Pages:     make([]ExtractedPage, 0, pageCount),
	}

	dpi := e.DPI
	if dpi <= 0 {
		dpi = 144
	}

	for pageIndex := 0; pageIndex < pageCount; pageIndex++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		imagePNG, err := doc.ImagePNG(pageIndex, dpi)
		if err != nil {
			return nil, fmt.Errorf("render page %d: %w", pageIndex+1, err)
		}

		bounds, err := doc.Bound(pageIndex)
		if err != nil {
			return nil, fmt.Errorf("read page %d bounds: %w", pageIndex+1, err)
		}

		text, err := doc.Text(pageIndex)
		if err != nil {
			return nil, fmt.Errorf("extract page %d text: %w", pageIndex+1, err)
		}

		result.Pages = append(result.Pages, ExtractedPage{
			PageIndex:   pageIndex,
			Width:       float64(bounds.Dx()),
			Height:      float64(bounds.Dy()),
			ImageBase64: base64.StdEncoding.EncodeToString(imagePNG),
			Blocks:      textBlocks(pageIndex, text),
		})
	}

	return result, nil
}

func textBlocks(pageIndex int, text string) []ExtractedBlock {
	lines := strings.Split(text, "\n")
	blocks := make([]ExtractedBlock, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		blockIndex := len(blocks)
		blocks = append(blocks, ExtractedBlock{
			BlockID:   fmt.Sprintf("p%03d-b%03d", pageIndex, blockIndex),
			PageIndex: pageIndex,
			BlockType: "text",
			Text:      line,
			LineCount: 1,
		})
	}
	return blocks
}
