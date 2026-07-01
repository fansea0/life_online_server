package ai

import (
	"testing"
	"time"

	"github.com/volcengine/volcengine-go-sdk/service/arkruntime/model"
	"life-online/pkg/eino"
)

func TestBuildArkChatModelConfigDisablesThinkingAndLimitsWait(t *testing.T) {
	cfg := buildArkChatModelConfig(eino.ModelConfig{
		ModelName:   "model",
		ModelApiKey: "key",
	})

	if cfg.Thinking == nil || cfg.Thinking.Type != model.ThinkingTypeDisabled {
		t.Fatalf("Thinking = %#v, want disabled", cfg.Thinking)
	}
	if cfg.Timeout == nil || *cfg.Timeout != time.Minute {
		t.Fatalf("Timeout = %v, want 1m", cfg.Timeout)
	}
	if cfg.RetryTimes == nil || *cfg.RetryTimes != 0 {
		t.Fatalf("RetryTimes = %v, want 0", cfg.RetryTimes)
	}
}
