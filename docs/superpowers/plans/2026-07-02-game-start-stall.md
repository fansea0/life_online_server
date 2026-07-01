# Game Start Stall Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Ensure audio initialization and Ark deep thinking can no longer leave the game indefinitely stuck before the first visible story.

**Architecture:** Add a small browser-side timeout utility so audio remains best-effort, and centralize Ark request policy in a testable config builder that disables thinking and limits request duration. Keep the WebSocket protocol and story rendering unchanged.

**Tech Stack:** Go 1.24, CloudWeGo Eino Ark model, Gin/WebSocket, browser JavaScript, Node built-in test runner.

---

## File Structure

- Create `resource/static/audio_guard.js`: reusable promise timeout boundary for browser media initialization.
- Create `resource/static/audio_guard.test.js`: Node tests for success, rejection, and never-settling tasks.
- Modify `resource/static/sanguo.html`: load the utility and guard every awaited audio-unlock operation.
- Create `service/ai/ai_test.go`: verify the Ark policy applied by the model factory.
- Modify `service/ai/ai.go`: build an Ark config with disabled thinking, zero retries, and a finite timeout.
- Create `service/game/game_test.go`: verify the initial context receives a user start instruction.
- Modify `service/game/game.go`: share the same initial user prompt between sync and streaming starts.

### Task 1: Add the audio timeout boundary

**Files:**
- Create: `resource/static/audio_guard.test.js`
- Create: `resource/static/audio_guard.js`

- [ ] **Step 1: Write the failing tests**

```javascript
const test = require('node:test');
const assert = require('node:assert/strict');
const { settleWithin } = require('./audio_guard.js');

test('settleWithin returns true when the task completes', async () => {
    assert.equal(await settleWithin(() => Promise.resolve(), 20), true);
});

test('settleWithin returns false when the task rejects', async () => {
    assert.equal(await settleWithin(() => Promise.reject(new Error('blocked')), 20), false);
});

test('settleWithin returns false when the task never settles', async () => {
    const startedAt = Date.now();
    assert.equal(await settleWithin(() => new Promise(() => {}), 20), false);
    assert.ok(Date.now() - startedAt < 200);
});
```

- [ ] **Step 2: Run the test and verify RED**

Run: `node --test resource/static/audio_guard.test.js`

Expected: FAIL because `resource/static/audio_guard.js` does not exist.

- [ ] **Step 3: Implement the minimal utility**

```javascript
(function (root, factory) {
    const api = factory();
    if (typeof module === 'object' && module.exports) {
        module.exports = api;
    }
    root.AudioGuard = api;
})(typeof globalThis !== 'undefined' ? globalThis : this, function () {
    async function settleWithin(task, timeoutMs) {
        return Promise.race([
            Promise.resolve().then(task).then(() => true, () => false),
            new Promise(resolve => setTimeout(() => resolve(false), timeoutMs))
        ]);
    }

    return { settleWithin };
});
```

- [ ] **Step 4: Run the test and verify GREEN**

Run: `node --test resource/static/audio_guard.test.js`

Expected: 3 tests pass.

### Task 2: Prevent audio from blocking the start message

**Files:**
- Modify: `resource/static/sanguo.html:8-15`
- Modify: `resource/static/sanguo.html:137-192`

- [ ] **Step 1: Load the timeout utility before the game script**

Add:

```html
<script src="audio_guard.js"></script>
```

- [ ] **Step 2: Guard AudioContext and media element promises**

Use an 800 ms boundary:

```javascript
const audioUnlockTimeoutMs = 800;
const resumed = await AudioGuard.settleWithin(
    () => audioContext.resume(),
    audioUnlockTimeoutMs
);
if (!resumed) {
    console.warn('unlockAudioOnce: Web Audio API unlock timed out or failed');
}
```

For each media element, run the play/pause/reset sequence through `AudioGuard.settleWithin`. Set `STATE.audioUnlocked = true` in `finally` so no media failure can block `startExplore`.

- [ ] **Step 3: Run the audio utility tests**

Run: `node --test resource/static/audio_guard.test.js`

Expected: 3 tests pass.

### Task 3: Lock down the Ark request policy

**Files:**
- Create: `service/ai/ai_test.go`
- Modify: `service/ai/ai.go`

- [ ] **Step 1: Write the failing Ark config test**

```go
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
```

- [ ] **Step 2: Run the test and verify RED**

Run: `/Users/fansea/sdk/go1.24.3/bin/go test ./service/ai -run TestBuildArkChatModelConfigDisablesThinkingAndLimitsWait -count=1`

Expected: FAIL because `buildArkChatModelConfig` does not exist.

- [ ] **Step 3: Implement the minimal Ark config builder**

Add constants and a focused builder:

```go
const modelRequestTimeout = time.Minute

func buildArkChatModelConfig(modelCfg eino.ModelConfig) *ark.ChatModelConfig {
    retryTimes := 0
    timeout := modelRequestTimeout
    return &ark.ChatModelConfig{
        APIKey:      modelCfg.ModelApiKey,
        Region:      "cn-wuhan",
        Model:       modelCfg.ModelName,
        Timeout:     &timeout,
        RetryTimes:  &retryTimes,
        Thinking:    &model.Thinking{Type: model.ThinkingTypeDisabled},
        Temperature: volcengine.Float32(1.0),
        TopP:        volcengine.Float32(1.0),
    }
}
```

Make `GetArkTextModel` call this builder and retain its existing error return behavior.

- [ ] **Step 4: Run the focused test and verify GREEN**

Run: `/Users/fansea/sdk/go1.24.3/bin/go test ./service/ai -run TestBuildArkChatModelConfigDisablesThinkingAndLimitsWait -count=1`

Expected: PASS.

### Task 4: Add the initial user start instruction

**Files:**
- Create: `service/game/game_test.go`
- Modify: `service/game/game.go`

- [x] **Step 1: Write and run the failing context test**

Verify that `appendInitialGamePrompt` appends a `schema.User` message containing the fixed start instruction.

Run: `/Users/fansea/sdk/go1.24.3/bin/go test ./service/game -run TestAppendInitialGamePromptAddsUserInstruction -count=1`

Expected before implementation: FAIL because the helper and prompt constant do not exist.

- [x] **Step 2: Implement and verify the shared context path**

Append the fixed user instruction in both `StartGame` and `StartGameStream`, then rerun the focused test.

Expected after implementation: PASS.

### Task 5: Verify the complete fix

**Files:**
- Verify only; no new production files.

- [ ] **Step 1: Format and run automated tests**

Run:

```bash
gofmt -w service/ai/ai.go service/ai/ai_test.go
/Users/fansea/sdk/go1.24.3/bin/go test ./...
node --test resource/static/audio_guard.test.js
```

Expected: all tests pass.

- [ ] **Step 2: Build and restart the local game**

Build the current source for port `18080` without retaining a source diff for the port override, stop the previous playtest process, and start the new binary.

Expected: `GET http://127.0.0.1:18080/` returns HTTP 200.

- [ ] **Step 3: Verify through browser developer tooling**

Reload `sanguo.html`, select a role, click “开始探索”, and inspect CDP WebSocket events.

Expected:

- a `Network.webSocketFrameSent` event containing `type:"start"`;
- received `session`, `content`, and `end` frames;
- three visible choice buttons;
- selecting one choice produces another content/end cycle;
- no permanent “对方正在输入...” state.

- [ ] **Step 4: Review the final diff**

Run: `git diff --check && git diff -- service/ai/ai.go service/ai/ai_test.go resource/static/sanguo.html resource/static/audio_guard.js resource/static/audio_guard.test.js`

Expected: no whitespace errors and only the approved fix is present.
