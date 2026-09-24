package llm

import (
	"strings"
	"testing"

	"orchestrator/internal/application"
)

func TestReadHiveStream_FoldsChunksAndReportsProgress(t *testing.T) {
	sse := `data: {"model":"m","choices":[{"delta":{"reasoning_content":"think","content":""}}]}

data: {"model":"m","choices":[{"delta":{"content":"Hel"}}]}

data: {"model":"m","choices":[{"delta":{"content":"lo"},"finish_reason":"stop"}]}

data: {"choices":[],"usage":{"prompt_tokens":5,"completion_tokens":9,"completion_tokens_details":{"reasoning_tokens":4}}}

data: [DONE]

`
	var last application.ChatProgress
	got, err := readHiveStream(strings.NewReader(sse), func(p application.ChatProgress) { last = p })
	if err != nil {
		t.Fatal(err)
	}
	if c := got.Choices[0].Message.Content; c != "Hello" {
		t.Fatalf("content = %q", c)
	}
	if got.Choices[0].FinishReason != "stop" || got.Model != "m" {
		t.Fatalf("finish/model = %q/%q", got.Choices[0].FinishReason, got.Model)
	}
	if got.Usage.PromptTokens != 5 || got.Usage.reasoning() != 4 {
		t.Fatalf("usage = %+v", got.Usage)
	}
	if last.ReasoningChars != 5 || last.ContentChars != 5 {
		t.Fatalf("progress = %+v", last)
	}
}
