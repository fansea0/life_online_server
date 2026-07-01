package game

import (
	"testing"

	"github.com/cloudwego/eino/schema"
)

func TestAppendInitialGamePromptAddsUserInstruction(t *testing.T) {
	context := []*schema.Message{schema.SystemMessage("system")}

	got := appendInitialGamePrompt(context)

	if len(got) != 2 {
		t.Fatalf("len(context) = %d, want 2", len(got))
	}
	if got[1].Role != schema.User {
		t.Fatalf("role = %q, want %q", got[1].Role, schema.User)
	}
	if got[1].Content != initialGamePrompt {
		t.Fatalf("content = %q, want %q", got[1].Content, initialGamePrompt)
	}
}
