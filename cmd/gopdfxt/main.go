package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/kiry163/easyllm"
	"github.com/kiry163/gopdfxt"
)

func main() {
	input := flag.String("input", "", "path to input PDF")
	output := flag.String("output", "", "optional text output path")
	resultPath := flag.String("result", "result.json", "optional JSON result output path")
	provider := flag.String("provider", gopdfxt.ProviderQwen, "LLM provider")
	model := flag.String("model", "qwen3-vl-plus", "LLM model")
	apiKey := flag.String("api-key", os.Getenv("GOPDFXT_API_KEY"), "LLM API key")
	debugDir := flag.String("debug-dir", "", "optional directory for debug artifacts")
	flag.Parse()

	if *input == "" {
		fmt.Fprintln(os.Stderr, "-input is required")
		os.Exit(2)
	}

	opts := gopdfxt.Options{
		LLM:          llmOptions(*provider, *apiKey, *model),
		Concurrency:  30,
		AllowPartial: true,
		Hooks:        progressHooks(os.Stderr),
		LLMHooks:     debugHooks(*debugDir),
	}
	if *debugDir != "" {
		opts.Extractor = debugExtractor{
			next: gopdfxt.NewDefaultExtractor(),
			dir:  *debugDir,
		}
	}

	converter, err := gopdfxt.New(opts)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	result, err := converter.ConvertFile(context.Background(), *input)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if *resultPath != "" {
		if err := writeJSON(*resultPath, result); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}

	outputText := formatResult(result)
	if *output == "" {
		fmt.Print(outputText)
		return
	}
	if err := os.WriteFile(*output, []byte(outputText), 0o644); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func llmOptions(provider, apiKey, model string) gopdfxt.LLMOptions {
	enableThinking := false
	return gopdfxt.LLMOptions{
		Provider:       provider,
		APIKey:         apiKey,
		Model:          model,
		EnableThinking: &enableThinking,
	}
}

func progressHooks(out *os.File) gopdfxt.Hooks {
	return gopdfxt.Hooks{
		OnPageDone: func(ctx context.Context, e gopdfxt.PageDoneEvent) {
			fmt.Fprintln(out, formatProgress(e.PageIndex+1, e.PageCount))
		},
		OnPageError: func(ctx context.Context, e gopdfxt.PageErrorEvent) {
			fmt.Fprintf(out, "page failed: %d: %v\n", e.PageIndex+1, e.Err)
		},
	}
}

func formatProgress(done, total int) string {
	return fmt.Sprintf("processed pages: %d/%d", done, total)
}

type debugExtractor struct {
	next gopdfxt.Extractor
	dir  string
}

func (e debugExtractor) Extract(ctx context.Context, input gopdfxt.PDFInput) (*gopdfxt.ExtractedDocument, error) {
	doc, err := e.next.Extract(ctx, input)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(e.dir, 0o755); err != nil {
		return nil, err
	}
	if err := writeJSON(filepath.Join(e.dir, "extracted.json"), compactExtractedDocument(doc)); err != nil {
		return nil, err
	}
	pagesDir := filepath.Join(e.dir, "pages")
	if err := os.MkdirAll(pagesDir, 0o755); err != nil {
		return nil, err
	}
	for _, page := range doc.Pages {
		if page.ImageBase64 == "" {
			continue
		}
		imageBytes, err := base64.StdEncoding.DecodeString(page.ImageBase64)
		if err != nil {
			return nil, err
		}
		pagePath := filepath.Join(pagesDir, fmt.Sprintf("page-%03d.png", page.PageIndex+1))
		if err := os.WriteFile(pagePath, imageBytes, 0o644); err != nil {
			return nil, err
		}
	}
	return doc, nil
}

func debugHooks(dir string) easyllm.Hooks {
	if dir == "" {
		return easyllm.Hooks{}
	}
	return easyllm.Hooks{
		OnModelRequest: func(e easyllm.ModelRequestEvent) error {
			return appendJSONL(filepath.Join(dir, "model_requests.jsonl"), e.Request)
		},
		OnModelResponse: func(e easyllm.ModelResponseEvent) error {
			return appendJSONL(filepath.Join(dir, "model_responses.jsonl"), e.Response)
		},
		OnRunFinish: func(e easyllm.RunFinishEvent) error {
			return appendJSONL(filepath.Join(dir, "run_finishes.jsonl"), e)
		},
	}
}

func writeJSON(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o644)
}

func appendJSONL(path string, value any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	data = append(data, '\n')
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = file.Write(data)
	return err
}

func compactExtractedDocument(doc *gopdfxt.ExtractedDocument) any {
	type compactPage struct {
		PageIndex int                      `json:"page_index"`
		Width     float64                  `json:"width"`
		Height    float64                  `json:"height"`
		Blocks    []gopdfxt.ExtractedBlock `json:"blocks"`
	}
	type compactDocument struct {
		PDF       string        `json:"pdf"`
		PageCount int           `json:"page_count"`
		Pages     []compactPage `json:"pages"`
	}
	if doc == nil {
		return compactDocument{}
	}
	result := compactDocument{
		PDF:       doc.PDF,
		PageCount: doc.PageCount,
		Pages:     make([]compactPage, 0, len(doc.Pages)),
	}
	for _, page := range doc.Pages {
		result.Pages = append(result.Pages, compactPage{
			PageIndex: page.PageIndex,
			Width:     page.Width,
			Height:    page.Height,
			Blocks:    page.Blocks,
		})
	}
	return result
}

func formatResult(result *gopdfxt.Result) string {
	if result == nil || len(result.Articles) == 0 {
		return ""
	}

	out := ""
	for i, article := range result.Articles {
		if i > 0 {
			out += "\n"
		}
		if article.Title != "" {
			out += article.Title + "\n\n"
		}
		out += article.Content
	}
	return out
}
