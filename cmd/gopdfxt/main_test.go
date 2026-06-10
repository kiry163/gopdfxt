package main

import "testing"

func TestLLMOptionsDefaultDisableThinking(t *testing.T) {
	opts := llmOptions("qwen", "key", "model")
	if opts.EnableThinking == nil {
		t.Fatalf("expected EnableThinking to be set")
	}
	if *opts.EnableThinking {
		t.Fatalf("expected thinking mode to be disabled by default")
	}
}

func TestFormatProgress(t *testing.T) {
	got := formatProgress(2, 5)
	want := "processed pages: 2/5"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}
