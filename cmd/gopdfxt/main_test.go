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
