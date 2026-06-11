package main

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/kiry163/gopdfxt"
)

func TestLLMOptionsDefaultDisableThinking(t *testing.T) {
	opts := llmOptions("qwen", "key", "model", 0, 0)
	if opts.EnableThinking == nil {
		t.Fatalf("expected EnableThinking to be set")
	}
	if *opts.EnableThinking {
		t.Fatalf("expected thinking mode to be disabled by default")
	}
}

func TestLLMOptionsSetsMaxRetries(t *testing.T) {
	opts := llmOptions("qwen", "key", "model", 3, 0)
	if opts.MaxRetries != 3 {
		t.Fatalf("expected max retries to be 3, got %d", opts.MaxRetries)
	}
}

func TestLLMOptionsSetsRequestTimeout(t *testing.T) {
	opts := llmOptions("qwen", "key", "model", 3, 120*time.Second)
	if opts.Timeout != 120*time.Second {
		t.Fatalf("expected request timeout to be 120s, got %s", opts.Timeout)
	}
}

func TestFormatProgress(t *testing.T) {
	got := formatProgress(2, 5)
	want := "processed pages: 2/5"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestResultJSONIncludesContentFailuresAndDetails(t *testing.T) {
	result := &gopdfxt.Result{
		Articles:    []gopdfxt.Article{{Title: "Title", Content: "Body\n", Pages: gopdfxt.PageRange{Start: 0, End: 0}}},
		FailedPages: []gopdfxt.PageError{{PageIndex: 1, Error: "model failed"}},
		Details: gopdfxt.ProcessingDetails{
			PageCount:      2,
			SucceededPages: 1,
			FailedPages:    1,
			ModelCalls:     1,
			Retries:        0,
		},
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("Marshal returned error: %v", err)
	}
	got := string(data)
	for _, want := range []string{"articles", "content", "failed_pages", "details", "model failed"} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected JSON to contain %q, got %s", want, got)
		}
	}
}
