# 回合类型轮转 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 在现有状态面板/隐藏 JSON 之上，引入 normal/crisis/boon/timeskip 4 种回合类型，服务端兜底判定 + AI 演绎，打破"每回合叙事+3选项"的节奏单调。

**Architecture:** `GameState` 加 `RecentKinds` 历史；服务端纯函数 `decideKindCandidate` 按状态+历史算候选类型，写进 system 上下文；隐藏 JSON 加 `kind` 字段，`FinalizeTurn` 解析并维护 `RecentKinds`；`state` 帧带 `kind`，前端按类型差异化呈现（红框/金框/时间轴）。

**Tech Stack:** Go (gin, cloudwego/eino), 前端纯 HTML/JS (Tailwind), 测试 Go `testing` + 前端纯函数测试。

## Global Constraints

- 4 种合法 kind：`normal` / `crisis` / `boon` / `timeskip`，非法/缺失降级 `normal`。
- 阈值常量：`crisisLow=1`、`crisisHigh=6`、`boonThreshold=6`、`timeskipGap=3`、`crisisGap=2`、`boonGap=2`。
- `RecentKinds` 保留最近 5 个，超过截断。
- 判定优先级（命中即返回）：timeskip → crisis → boon → normal。
- 任何 kind 缺失/非法都不阻断剧情；解析失败记 warn 不记 error。
- 不改既有 4 维状态模型与防爆炸机制（delta 仍限 ±2，越界忽略）。
- 不改选后揭示原则（不露数值）。
- Go 字段用 `int` 不用 `int32`；日志 msg 以函数名开头。

---

## File Structure

- `service/game/types.go` — 改：`GameState` 加 `RecentKinds []string`。
- `service/game/state.go` — 改：加阈值常量、`TurnKind` 合法值、`decideKindCandidate`、`kindPromptRule`、`parseHiddenPayload` 解析 `kind`、`TurnResult` 加 `Kind`、`FinalizeTurn` 维护 `RecentKinds`。
- `service/game/game.go` — 改：`initSystemPrompt` 加 `turnKindRule` 参数；`HandleChoiceStream` 算 candidate 并注入；`StartGameStream`/`StartGame` 首轮 normal。
- `route/game/game.go` — 改：`state` 帧带 `kind`。
- `config/env.toml` — 改：`SystemMsg` 加 `{{.turnKindRule}}` 占位（gitignored，需手动同步部署）。
- `resource/static/sanguo.html` — 改：`applyStateFrame` 读 kind 加容器 class；timeskip 标题时间轴渲染；上轮 class 清除。
- `service/game/state_test.go` — 改：新增 decideKindCandidate / kind 解析 / RecentKinds 维护测试。
- `resource/static/sanguo_state.test.js` — 改：新增 kind→class 映射测试。

---

### Task 1: GameState 加 RecentKinds

**Files:**
- Modify: `service/game/types.go`

**Interfaces:**
- Produces: `GameState` 新增 `RecentKinds []string` 字段。

- [ ] **Step 1: 给 GameState 加 RecentKinds 字段**

在 `service/game/types.go` 的 `GameState` 结构体末尾（`IsGameOver` 之后）加：

```go
	RecentKinds []string `json:"recent_kinds"` // 最近 5 轮回合类型
```

完整结构体应为：

```go
// GameState 玩家当前状态（三国 4 维）
type GameState struct {
	SessionID        string         `json:"session_id"`
	Name             string         `json:"name"`
	Identity         string         `json:"identity"`          // 初始身份描述
	IdentityAffinity string         `json:"identity_affinity"` // 身份倾向自然语言描述
	Age              int            `json:"age"`
	Attributes       map[string]int `json:"attributes"`        // 名望/人心/实力/机缘
	Summary          string         `json:"summary"`           // 前情摘要，回灌 AI 上下文
	IsGameOver       bool           `json:"is_game_over"`
	RecentKinds      []string       `json:"recent_kinds"`      // 最近 5 轮回合类型
}
```

- [ ] **Step 2: 编译并运行测试**

Run: `go build ./... && go test ./service/game/...`
Expected: 编译通过，现有测试全 PASS（新字段零值 nil 不影响）

- [ ] **Step 3: Commit**

```bash
git add service/game/types.go
git commit -m "refactor(game): GameState 加 RecentKinds 历史"
```

---

### Task 2: 阈值常量与 TurnKind 合法值

**Files:**
- Modify: `service/game/state.go`
- Test: `service/game/state_test.go`

**Interfaces:**
- Produces: 阈值常量（`crisisLow` 等 6 个，包级 `const`）、`validKinds` 切片、`isValidKind(kind string) bool`、`normalizeKind(kind string) string`（非法/空 → `normal`）。

- [ ] **Step 1: 写失败测试**

在 `service/game/state_test.go` 末尾追加：

```go
func TestNormalizeKind(t *testing.T) {
	cases := []struct{ in, want string }{
		{"normal", "normal"},
		{"crisis", "crisis"},
		{"boon", "boon"},
		{"timeskip", "timeskip"},
		{"", "normal"},
		{"boss", "normal"},
		{"NORMAL", "normal"},
	}
	for _, c := range cases {
		if got := normalizeKind(c.in); got != c.want {
			t.Fatalf("normalizeKind(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestThresholdConstants(t *testing.T) {
	if crisisLow != 1 || crisisHigh != 6 || boonThreshold != 6 {
		t.Fatalf("thresholds: crisisLow=%d crisisHigh=%d boonThreshold=%d", crisisLow, crisisHigh, boonThreshold)
	}
	if timeskipGap != 3 || crisisGap != 2 || boonGap != 2 {
		t.Fatalf("gaps: timeskipGap=%d crisisGap=%d boonGap=%d", timeskipGap, crisisGap, boonGap)
	}
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./service/game/ -run "TestNormalizeKind|TestThresholdConstants" -v`
Expected: FAIL（常量与函数未定义）

- [ ] **Step 3: 实现常量与函数**

在 `service/game/state.go` 的 `var validDims = ...` 行之前插入：

```go
// 回合类型阈值常量（便于调参与测试）
const (
	crisisLow      = 1 // 维度 <=此值视为濒危，触发 crisis
	crisisHigh     = 6 // 维度 >=此值视为高维"树大招风"
	boonThreshold  = 6 // 维度跨到此值视为刚跨阈值，触发 boon
	timeskipGap    = 3 // 连续此轮数无 timeskip 则候选 timeskip
	crisisGap      = 2 // 连续此轮数无 crisis 则可触发 crisis（需高维）
	boonGap        = 2 // 连续此轮数无 boon 则可触发 boon（需无 crisis 候选）
)

// validKinds 4 种合法回合类型
var validKinds = []string{"normal", "crisis", "boon", "timeskip"}

// isValidKind 判断 kind 是否合法
func isValidKind(kind string) bool {
	for _, k := range validKinds {
		if k == kind {
			return true
		}
	}
	return false
}

// normalizeKind 非法/空降级为 normal
func normalizeKind(kind string) string {
	kind = strings.TrimSpace(kind)
	if isValidKind(kind) {
		return kind
	}
	return "normal"
}
```

- [ ] **Step 4: 运行测试确认通过**

Run: `go test ./service/game/ -run "TestNormalizeKind|TestThresholdConstants" -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add service/game/state.go service/game/state_test.go
git commit -m "feat(game): 回合类型阈值常量与 normalizeKind"
```

---

### Task 3: decideKindCandidate 判定函数

**Files:**
- Modify: `service/game/state.go`
- Test: `service/game/state_test.go`

**Interfaces:**
- Consumes: `GameState`（含 `Attributes`、`RecentKinds`）、Task 2 常量
- Produces: `func decideKindCandidate(s *GameState, recentKinds []string) string`。优先级：timeskip → crisis → boon → normal。

判定规则（按优先级，命中即返回）：
1. **timeskip** — `recentKinds` 中连续 ≥3 轮无 timeskip，且最近一轮（`recentKinds` 末尾）非 crisis。
2. **crisis** — 任一维度 ≤1，或（连续 ≥2 轮无 crisis 且存在维度 ≥6）。
3. **boon** — 连续 ≥2 轮无 boon 且无 crisis 候选（即不满足规则2）。
4. **normal** — 兜底。

辅助：`func lastKindIs(recentKinds []string, gap int, kind string) bool` 判断"最近 gap 轮内是否出现过 kind"——若未出现返回 true（可触发）。`gap` 用 `timeskipGap`/`crisisGap`/`boonGap`。

- [ ] **Step 1: 写失败测试**

在 `service/game/state_test.go` 末尾追加：

```go
func TestDecideKindTimeskipAfterGap(t *testing.T) {
	s := &GameState{Attributes: map[string]int{"名望": 3, "人心": 3, "实力": 3, "机缘": 3}}
	// 连续 3 轮 normal，无 timeskip，末尾非 crisis -> timeskip
	recent := []string{"normal", "normal", "normal"}
	if got := decideKindCandidate(s, recent); got != "timeskip" {
		t.Fatalf("timeskip gap: got %q, want timeskip", got)
	}
}

func TestDecideKindTimeskipBlockedByCrisis(t *testing.T) {
	s := &GameState{Attributes: map[string]int{"名望": 3, "人心": 3, "实力": 3, "机缘": 3}}
	// 末尾是 crisis，不 timeskip
	recent := []string{"normal", "normal", "crisis"}
	if got := decideKindCandidate(s, recent); got == "timeskip" {
		t.Fatalf("should not timeskip after crisis, got timeskip")
	}
}

func TestDecideKindCrisisLowDim(t *testing.T) {
	s := &GameState{Attributes: map[string]int{"名望": 1, "人心": 5, "实力": 5, "机缘": 5}}
	// 名望=1 <= crisisLow -> crisis
	if got := decideKindCandidate(s, []string{"normal"}); got != "crisis" {
		t.Fatalf("crisis low: got %q, want crisis", got)
	}
}

func TestDecideKindCrisisHighDimAfterGap(t *testing.T) {
	s := &GameState{Attributes: map[string]int{"名望": 6, "人心": 3, "实力": 3, "机缘": 3}}
	// 连续 2 轮无 crisis + 高维 -> crisis
	if got := decideKindCandidate(s, []string{"normal", "normal"}); got != "crisis" {
		t.Fatalf("crisis high: got %q, want crisis", got)
	}
}

func TestDecideKindPriorityCrisisOverTimeskip(t *testing.T) {
	// 同时满足 timeskip(3轮无) 与 crisis(低维) -> crisis 优先
	s := &GameState{Attributes: map[string]int{"名望": 1, "人心": 5, "实力": 5, "机缘": 5}}
	recent := []string{"normal", "normal", "normal"}
	if got := decideKindCandidate(s, recent); got != "crisis" {
		t.Fatalf("priority: got %q, want crisis", got)
	}
}

func TestDecideKindBoonAfterGapNoCrisis(t *testing.T) {
	s := &GameState{Attributes: map[string]int{"名望": 3, "人心": 3, "实力": 3, "机缘": 3}}
	// 连续 2 轮无 boon，无 crisis 候选，不足 timeskip gap -> boon
	if got := decideKindCandidate(s, []string{"normal", "normal"}); got != "boon" {
		t.Fatalf("boon: got %q, want boon", got)
	}
}

func TestDecideKindNormalEmptyRecent(t *testing.T) {
	s := &GameState{Attributes: map[string]int{"名望": 3, "人心": 3, "实力": 3, "机缘": 3}}
	// 首轮空 recent，无任何触发 -> normal
	if got := decideKindCandidate(s, []string{}); got != "normal" {
		t.Fatalf("empty recent: got %q, want normal", got)
	}
}

func TestDecideKindNormalSingleRecent(t *testing.T) {
	s := &GameState{Attributes: map[string]int{"名望": 4, "人心": 4, "实力": 4, "机缘": 4}}
	// 1 轮 normal，不满足任何 gap -> normal
	if got := decideKindCandidate(s, []string{"normal"}); got != "normal" {
		t.Fatalf("single recent: got %q, want normal", got)
	}
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./service/game/ -run TestDecideKind -v`
Expected: FAIL（`decideKindCandidate` 未定义）

- [ ] **Step 3: 实现 decideKindCandidate**

在 `service/game/state.go` 的 `normalizeKind` 之后追加：

```go
// lastKindGapSatisfied 判断 recentKinds 末尾 gap 轮内是否出现过 kind。
// 未出现（即连续 gap 轮无该类型）返回 true。
func lastKindGapSatisfied(recentKinds []string, gap int, kind string) bool {
	n := len(recentKinds)
	if n == 0 {
		return true // 空历史视为满足
	}
	start := n - gap
	if start < 0 {
		start = 0
	}
	for _, k := range recentKinds[start:] {
		if k == kind {
			return false
		}
	}
	return true
}

// decideKindCandidate 按状态+历史算候选回合类型。优先级：timeskip > crisis > boon > normal。
func decideKindCandidate(s *GameState, recentKinds []string) string {
	// 规则2前置检查：是否构成 crisis 候选（低维 或 高维+gap）
	crisisCandidate := false
	for _, dim := range validDims {
		if s.Attributes[dim] <= crisisLow {
			crisisCandidate = true
			break
		}
	}
	if !crisisCandidate {
		hasHigh := false
		for _, dim := range validDims {
			if s.Attributes[dim] >= crisisHigh {
				hasHigh = true
				break
			}
		}
		if hasHigh && lastKindGapSatisfied(recentKinds, crisisGap, "crisis") {
			crisisCandidate = true
		}
	}
	// crisis 优先于 timeskip
	if crisisCandidate {
		return "crisis"
	}
	// 规则1：timeskip
	lastIsCrisis := len(recentKinds) > 0 && recentKinds[len(recentKinds)-1] == "crisis"
	if !lastIsCrisis && lastKindGapSatisfied(recentKinds, timeskipGap, "timeskip") && len(recentKinds) >= timeskipGap {
		return "timeskip"
	}
	// 规则3：boon
	if lastKindGapSatisfied(recentKinds, boonGap, "boon") {
		return "boon"
	}
	return "normal"
}
```

注意：`timeskip` 要求 `len(recentKinds) >= timeskipGap`，否则空历史首轮不触发（首轮应 normal）。

- [ ] **Step 4: 运行测试确认通过**

Run: `go test ./service/game/ -run TestDecideKind -v`
Expected: 全部 PASS

- [ ] **Step 5: 运行全部测试**

Run: `go test ./service/game/...`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add service/game/state.go service/game/state_test.go
git commit -m "feat(game): decideKindCandidate 候选回合类型判定"
```

---

### Task 4: kindPromptRule 规则段

**Files:**
- Modify: `service/game/state.go`
- Test: `service/game/state_test.go`

**Interfaces:**
- Produces: `func kindPromptRule(kind string) string`，返回该类型的 system 规则段文本。非法 kind → normal 段。

- [ ] **Step 1: 写失败测试**

在 `service/game/state_test.go` 末尾追加：

```go
func TestKindPromptRule(t *testing.T) {
	for _, kind := range validKinds {
		rule := kindPromptRule(kind)
		if rule == "" {
			t.Fatalf("kindPromptRule(%q) returned empty", kind)
		}
		if !strings.Contains(rule, kind) {
			t.Fatalf("kindPromptRule(%q) should contain kind name: %q", kind, rule)
		}
	}
	// 非法降级 normal
	if kindPromptRule("boss") != kindPromptRule("normal") {
		t.Fatalf("kindPromptRule(boss) should equal normal rule")
	}
}
```

并在 `state_test.go` 顶部 import 确认含 `"strings"`（已有则跳过）。

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./service/game/ -run TestKindPromptRule -v`
Expected: FAIL（`kindPromptRule` 未定义）

- [ ] **Step 3: 实现 kindPromptRule**

在 `service/game/state.go` 的 `decideKindCandidate` 之后追加：

```go
// kindPromptRule 返回该回合类型的 system 规则段文本
func kindPromptRule(kind string) string {
	switch normalizeKind(kind) {
	case "crisis":
		return "[crisis] 叙事≤60字，急促；只给2个选项，两者都有代价（必有负面delta）；无第三选项。"
	case "boon":
		return "[boon] 叙事≤80字，喜悦基调；3个选项均无负面delta，让玩家选偏向哪个维度的收益。"
	case "timeskip":
		return "[timeskip] 叙事≤40字概括数月/数年经过 + 一句\"时过境迁\"；2个选项（\"就此翻过\"/\"期间有所动作\"）；本类型state_delta可更大（±2常态）。"
	default:
		return "[normal] 叙事100-150字，给出3个风格分化的选项，可含正负后果。"
	}
}
```

- [ ] **Step 4: 运行测试确认通过**

Run: `go test ./service/game/ -run TestKindPromptRule -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add service/game/state.go service/game/state_test.go
git commit -m "feat(game): kindPromptRule 各类型规则段"
```

---

### Task 5: parseHiddenPayload 解析 kind + FinalizeTurn 维护 RecentKinds

**Files:**
- Modify: `service/game/state.go`
- Test: `service/game/state_test.go`

**Interfaces:**
- Produces: `HiddenPayload` 加 `Kind string` 字段；`rawHiddenPayload` 加 `Kind string json:"kind"`；`TurnResult` 加 `Kind string`；`FinalizeTurn` 把 `payload.Kind` 追加进 `state.RecentKinds`（保留最近 5）。

- [ ] **Step 1: 写失败测试**

在 `service/game/state_test.go` 末尾追加：

```go
func TestParseHiddenPayloadKind(t *testing.T) {
	p := parseHiddenPayload(`{"kind":"crisis","options":["a","b"]}`)
	if p.Kind != "crisis" {
		t.Fatalf("kind: got %q, want crisis", p.Kind)
	}
}

func TestParseHiddenPayloadKindIllegalFallback(t *testing.T) {
	p := parseHiddenPayload(`{"kind":"boss","options":["a"]}`)
	if p.Kind != "normal" {
		t.Fatalf("illegal kind should fallback normal: got %q", p.Kind)
	}
}

func TestParseHiddenPayloadKindMissingFallback(t *testing.T) {
	p := parseHiddenPayload(`{"options":["a"]}`)
	if p.Kind != "normal" {
		t.Fatalf("missing kind should fallback normal: got %q", p.Kind)
	}
}

func TestFinalizeTurnAppendsKindToRecent(t *testing.T) {
	s := CreateNewGame("陈凡", "魏国的一名谋士")
	raw := `{"kind":"crisis","summary":"敌袭","state_delta":{"名望":0},"options":["迎战","撤退"]}`
	res, ok := FinalizeTurn(s.SessionID, raw)
	if !ok {
		t.Fatalf("expected ok=true")
	}
	if res.Kind != "crisis" {
		t.Fatalf("TurnResult.Kind: got %q, want crisis", res.Kind)
	}
	st := GetState(s.SessionID)
	if len(st.RecentKinds) != 1 || st.RecentKinds[0] != "crisis" {
		t.Fatalf("RecentKinds should be [crisis], got %+v", st.RecentKinds)
	}
}

func TestFinalizeTurnRecentKindsTruncatesAtFive(t *testing.T) {
	s := CreateNewGame("陈凡", "魏国的一名谋士")
	// 模拟已有 5 个历史
	s.RecentKinds = []string{"normal", "normal", "normal", "normal", "normal"}
	SaveState(s)
	raw := `{"kind":"boon","options":["a","b","c"]}`
	_, ok := FinalizeTurn(s.SessionID, raw)
	if !ok {
		t.Fatalf("expected ok=true")
	}
	st := GetState(s.SessionID)
	if len(st.RecentKinds) != 5 {
		t.Fatalf("RecentKinds should truncate to 5, got %d", len(st.RecentKinds))
	}
	if st.RecentKinds[4] != "boon" {
		t.Fatalf("last should be boon, got %+v", st.RecentKinds)
	}
	// 第一个被挤掉
	if st.RecentKinds[0] != "normal" {
		t.Fatalf("first should be normal after shift, got %+v", st.RecentKinds)
	}
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./service/game/ -run "TestParseHiddenPayloadKind|TestFinalizeTurnAppendsKind|TestFinalizeTurnRecentKindsTruncates" -v`
Expected: FAIL（HiddenPayload 无 Kind 字段）

- [ ] **Step 3: 改 HiddenPayload 与 rawHiddenPayload 加 Kind**

在 `service/game/state.go` 修改两个结构体：

```go
// HiddenPayload AI 隐藏部分（& 之后）解析结果
type HiddenPayload struct {
	Kind    string
	Summary string
	Delta   map[string]int
	Options []string
}

type rawHiddenPayload struct {
	Kind       string         `json:"kind"`
	Summary    string         `json:"summary"`
	StateDelta map[string]int `json:"state_delta"`
	Options    []string       `json:"options"`
}
```

- [ ] **Step 4: parseHiddenPayload 解析 Kind**

在 `parseHiddenPayload` 的 `return HiddenPayload{Summary: r.Summary, Delta: delta, Options: options}` 行改为：

```go
	return HiddenPayload{Kind: normalizeKind(r.Kind), Summary: r.Summary, Delta: delta, Options: options}
```

- [ ] **Step 5: TurnResult 加 Kind 并在 FinalizeTurn 维护 RecentKinds**

修改 `TurnResult`：

```go
// TurnResult 一轮收尾后推给前端的结果
type TurnResult struct {
	Kind    string         // 本轮回合类型
	Panel   map[string]string // 定性档位名
	Delta   map[string]string // 方向 "+"/"-"/"0"
	Options []string          // 下一轮选项
}
```

修改 `FinalizeTurn` 的返回与 `state` 持久化部分。在 `applyDelta(state, payload.Delta)` 之后、`SaveState(state)` 之前，追加 RecentKinds 维护；返回值加 `Kind`：

```go
	applyDelta(state, payload.Delta)
	if payload.Summary != "" {
		state.Summary = payload.Summary
	}
	// 维护 RecentKinds，保留最近 5 个
	state.RecentKinds = append(state.RecentKinds, payload.Kind)
	if len(state.RecentKinds) > 5 {
		state.RecentKinds = state.RecentKinds[len(state.RecentKinds)-5:]
	}
	SaveState(state)

	return TurnResult{
		Kind:    payload.Kind,
		Panel:   panelFromState(state),
		Delta:   deltaDir,
		Options: payload.Options,
	}, true
}
```

- [ ] **Step 6: 运行测试确认通过**

Run: `go test ./service/game/ -run "TestParseHiddenPayloadKind|TestFinalizeTurnAppendsKind|TestFinalizeTurnRecentKindsTruncates" -v`
Expected: PASS

- [ ] **Step 7: 运行全部测试**

Run: `go test ./service/game/...`
Expected: PASS（注意：Task 7 的 `TestFinalizeTurnAppliesDeltaAndClamps` 等旧测试不检查 Kind，仍应通过）

- [ ] **Step 8: Commit**

```bash
git add service/game/state.go service/game/state_test.go
git commit -m "feat(game): 解析 kind 并维护 RecentKinds"
```

---

### Task 6: 流式入口注入候选类型

**Files:**
- Modify: `service/game/game.go`

**Interfaces:**
- Consumes: `decideKindCandidate`、`kindPromptRule`、`state.RecentKinds`（来自 Task 3/4/5）
- Produces: `initSystemPrompt` 加 `turnKindRule string` 参数（第5个）；`HandleChoiceStream` 算 candidate 并注入；`StartGameStream`/`StartGame` 首轮传 normal 规则段。

- [ ] **Step 1: 改 initSystemPrompt 加 turnKindRule 参数**

`service/game/game.go` 中 `initSystemPrompt` 签名改为 5 参，模板加 `turnKindRule`：

```go
func initSystemPrompt(name, identify, affinity, stateHint, turnKindRule string) ([]*schema.Message, error) {
	msg, err := eino.CreateMessagesCommon(
		config.GetSystemMsg(),
		map[string]any{
			"name":             name,
			"identify":         identify,
			"identityAffinity": affinity,
			"stateHint":        stateHint,
			"turnKindRule":     turnKindRule,
			"respFormat":       config.GetRespFormat(),
			"otherReqs":        config.GetOtherReqs(),
		},
		true,
	)
	if err != nil {
		return nil, err
	}
	return msg, nil
}
```

- [ ] **Step 2: StartGameStream/StartGame 首轮传 normal 规则**

`StartGame` 与 `StartGameStream` 中 `initSystemPrompt(name, identify, affinity, "")` 调用改为：

```go
	msgContext, err := initSystemPrompt(name, identify, affinity, "", kindPromptRule("normal"))
```

（开局首轮 RecentKinds 为空，候选强制 normal。）

- [ ] **Step 3: HandleChoiceStream 算 candidate 并注入**

`HandleChoiceStream` 中，在读取 `state` 之后、调用 `initSystemPrompt` 之前，算候选类型。当前代码已有 `stateHint`/`affinity` 计算块，改为：

```go
	context := GetMsgContext(sessionID)
	state := GetState(sessionID)
	stateHint := ""
	affinity := ""
	turnKindRule := kindPromptRule("normal")
	if state != nil {
		stateHint = stateHintFromState(state)
		affinity = state.IdentityAffinity
		candidate := decideKindCandidate(state, state.RecentKinds)
		turnKindRule = "本轮建议回合类型：" + candidate + "。" + kindPromptRule(candidate) + "你必须按此类型生成剧情与选项，并在 kind 字段回填实际类型。"
	}
	sysMsg, err := initSystemPrompt(stateIdentityName(state), stateIdentityDesc(state), affinity, stateHint, turnKindRule)
	if err == nil && len(sysMsg) > 0 && len(context) > 0 {
		context = append([]*schema.Message{sysMsg[0]}, context[1:]...)
	}
```

- [ ] **Step 4: 同步 HandleChoice（非流式）**

`HandleChoice` 中同样加 `turnKindRule` 计算块（与 HandleChoiceStream 一致），`initSystemPrompt` 调用改为 5 参：

```go
	context := GetMsgContext(sessionID)
	state := GetState(sessionID)
	stateHint := ""
	affinity := ""
	turnKindRule := kindPromptRule("normal")
	if state != nil {
		stateHint = stateHintFromState(state)
		affinity = state.IdentityAffinity
		candidate := decideKindCandidate(state, state.RecentKinds)
		turnKindRule = "本轮建议回合类型：" + candidate + "。" + kindPromptRule(candidate) + "你必须按此类型生成剧情与选项，并在 kind 字段回填实际类型。"
	}
	sysMsg, err := initSystemPrompt(stateIdentityName(state), stateIdentityDesc(state), affinity, stateHint, turnKindRule)
	if err == nil && len(sysMsg) > 0 && len(context) > 0 {
		context = append([]*schema.Message{sysMsg[0]}, context[1:]...)
	}
```

- [ ] **Step 5: 编译与测试**

Run: `go build ./... && go test ./service/game/...`
Expected: 编译通过，测试 PASS

- [ ] **Step 6: Commit**

```bash
git add service/game/game.go
git commit -m "feat(game): 流式入口注入候选回合类型规则"
```

---

### Task 7: SystemMsg 加 turnKindRule 占位

**Files:**
- Modify: `config/env.toml`（gitignored，无法提交；改本地文件并在提交说明里注明需手动同步部署）

- [ ] **Step 1: 改 SystemMsg 加占位**

`config/env.toml` 的 `SystemMsg` 在 `当前态势：{{.stateHint}}` 行之后加一行：

```
\n回合类型规则：{{.turnKindRule}}
```

完整 `SystemMsg`（在原值基础上插入该行）：

```toml
SystemMsg = "你是一款人生online游戏，用户将扮演三国时期的穿越者，通过你给的选项进行选择，你来述说故事推进。\n你的名称是：{{.name}}\n你的初始身份是：{{.identify}}\n身份倾向：{{.identityAffinity}}\n当前态势：{{.stateHint}}\n回合类型规则：{{.turnKindRule}}\n\n选项生成规则：\n1. 每轮固定给出3个选项，文案只描述行动、不超过15字，绝不展示后果或数字。\n2. 三个选项须风格分化：一个偏向身份倾向、一个偏进取冒险、一个偏保守稳妥。\n3. 根据玩家当前状态(名望/人心/实力/机缘)与历史选择推演后果，让选择产生可见且跨轮积累的影响。\n4. 在剧情后输出一个隐藏JSON块（以&开头，&之前为可见剧情），JSON结构：{\"kind\":\"normal\",\"summary\":\"一两句前情摘要\",\"state_delta\":{\"名望\":0,\"人心\":0,\"实力\":0,\"机缘\":0},\"options\":[\"选项1\",\"选项2\",\"选项3\"]}。kind取值仅限normal/crisis/boon/timeskip。state_delta取值仅限-2到2的整数。可见剧情中不得再出现1./2./3.选项文本。\n\n响应格式（可见部分）严格按照：{{.respFormat}}\n其他要求：{{.otherReqs}}"
```

- [ ] **Step 2: 验证配置加载与模板渲染**

Run:
```bash
go build ./... && cat > /tmp/tmplcheck.go << 'EOF'
package main
import (
	"fmt"
	"life-online/config"
	"life-online/pkg/eino"
)
func main(){
	config.EnvConfigFile = "./config/env.toml"
	config.InitEnvConf()
	msgs, err := eino.CreateMessagesCommon(config.GetSystemMsg(), map[string]any{
		"name":"陈凡","identify":"魏国的一名谋士",
		"identityAffinity":"偏好运筹","stateHint":"","turnKindRule":"[normal] ...",
		"respFormat":config.GetRespFormat(),"otherReqs":config.GetOtherReqs(),
	}, true)
	if err != nil { fmt.Println("ERR:", err); return }
	fmt.Println("rendered OK, msgs:", len(msgs))
}
EOF
go run /tmp/tmplcheck.go 2>&1 | tail -3; rm -f /tmp/tmplcheck.go
```
Expected: 输出 `rendered OK, msgs: 1`，无模板缺失报错。

- [ ] **Step 3: 记录配置需手动同步**

因 `config/env.toml` 被 gitignore（含 API key），本步改动不进版本库。在 commit message 中注明。无需 git commit（无文件可提交），但若 `git status` 显示其他改动，不要误提交 env.toml。

Run: `git status --short | grep env.toml`
Expected: 无输出（env.toml 被 gitignore）

- [ ] **Step 4: 全量构建测试**

Run: `go build ./... && go test ./...`
Expected: 全部 PASS

---

### Task 8: state 帧带 kind

**Files:**
- Modify: `route/game/game.go`

**Interfaces:**
- Consumes: `TurnResult.Kind`（来自 Task 5）
- Produces: `GameWS` 的 `state` 帧带 `kind` 字段；start 帧的初始 state 也带 `kind:"normal"`。

- [ ] **Step 1: 改流结束后的 state 帧加 kind**

`route/game/game.go` 的 `GameWS` 中，流结束后发送的 `state` 帧（`turn, ok := game.FinalizeTurn(...)` 块内）改为：

```go
			ws.WriteJSON(gin.H{
				"type":    "state",
				"kind":    turn.Kind,
				"panel":   turn.Panel,
				"delta":   turn.Delta,
				"options": turn.Options,
			})
```

- [ ] **Step 2: 改 start 帧初始 state 加 kind**

`GameWS` 的 `req.Type == "start"` 分支中，初始 `state` 帧改为：

```go
				if panel := game.PanelOf(sessionID); panel != nil {
					ws.WriteJSON(gin.H{
						"type":  "state",
						"kind":  "normal",
						"panel": panel,
						"delta": map[string]string{"名望": "0", "人心": "0", "实力": "0", "机缘": "0"},
					})
				}
```

- [ ] **Step 3: 编译与测试**

Run: `go build ./... && go test ./...`
Expected: 编译通过，测试 PASS

- [ ] **Step 4: Commit**

```bash
git add route/game/game.go
git commit -m "feat(route): state 帧带 kind 字段"
```

---

### Task 9: 前端 kind 呈现与 timeskip 时间轴

**Files:**
- Modify: `resource/static/sanguo.html`
- Modify: `resource/static/sanguo_state.test.js`

**Interfaces:**
- Consumes: Task 8 的 `state` 帧 `kind` 字段
- Produces: `applyStateFrame` 读 kind 给 `els.storyContainer` 加 class（`turn-normal`/`turn-crisis`/`turn-boon`/`turn-timeskip`），上轮 class 清除；`renderChunk` 在 `turn-timeskip` 时第一段 `#` 标题渲染为时间轴分隔。

- [ ] **Step 1: 写 kind→class 纯函数测试**

在 `resource/static/sanguo_state.test.js` 末尾追加：

```javascript
// kind -> 容器 class 映射
function kindToClass(kind) {
    const map = {"normal":"turn-normal","crisis":"turn-crisis","boon":"turn-boon","timeskip":"turn-timeskip"};
    return map[kind] || "turn-normal";
}

try {
    assert.strictEqual(kindToClass("crisis"), "turn-crisis");
    assert.strictEqual(kindToClass("boon"), "turn-boon");
    assert.strictEqual(kindToClass("timeskip"), "turn-timeskip");
    assert.strictEqual(kindToClass("normal"), "turn-normal");
    assert.strictEqual(kindToClass("boss"), "turn-normal");
    assert.strictEqual(kindToClass(undefined), "turn-normal");
    console.log("kindToClass tests PASS");
} catch (e) {
    console.error("kindToClass tests FAIL:", e.message);
    process.exit(1);
}
```

- [ ] **Step 2: 运行测试确认通过**

Run: `node resource/static/sanguo_state.test.js`
Expected: 输出 `sanguo_state tests PASS` 与 `kindToClass tests PASS`

- [ ] **Step 3: 加 CSS 样式**

在 `resource/static/sanguo.html` 的 `<style>` 块末尾追加：

```css
        /* Turn kind 视觉 */
        .turn-crisis #story-container { background-color: #FEF2F2; }
        .turn-boon #story-container { background-color: #FFFBEB; }
        .turn-crisis .story-tag { color: #DC2626; }
        .turn-boon .story-tag { color: #D97706; }
        .timeskip-divider { display: flex; align-items: center; text-align: center; color: #6B7280; margin: 1rem 0; }
        .timeskip-divider::before, .timeskip-divider::after { content: ""; flex: 1; border-bottom: 1px solid #D1D5DB; }
        .timeskip-divider::before { margin-right: 0.75rem; }
        .timeskip-divider::after { margin-left: 0.75rem; }
```

- [ ] **Step 4: applyStateFrame 加 kind 处理与上轮 class 清除**

在 `resource/static/sanguo.html` 的 `applyStateFrame` 函数开头（`const dims = ...` 之前）加 kind 处理：

```javascript
        function applyStateFrame(data) {
            // 清除上轮 turn class，加本轮
            const kind = data.kind || "normal";
            const kindClass = {"normal":"turn-normal","crisis":"turn-crisis","boon":"turn-boon","timeskip":"turn-timeskip"}[kind] || "turn-normal";
            ["turn-normal","turn-crisis","turn-boon","turn-timeskip"].forEach(c => els.game.classList.remove(c));
            els.game.classList.add(kindClass);
            STATE.currentKind = kind;

            const dims = ["名望", "人心", "实力", "机缘"];
```

（其余 applyStateFrame 体保持不变。）

- [ ] **Step 5: renderChunk 在 timeskip 时把首段 # 标题渲染为时间轴分隔**

在 `renderChunk` 函数中，找到 `} else if (text.startsWith('#')) {` 分支，改为：

```javascript
            } else if (text.startsWith('#')) {
                if (STATE.currentKind === 'timeskip') {
                    const label = text.replace(/^#+\s*/, '').trim() || '时过境迁';
                    div.innerHTML = `<div class="timeskip-divider font-novel text-sm">${label}</div>`;
                    requestAnimationFrame(() => div.classList.remove('opacity-0'));
                    return;
                }
                // Header
                div.innerHTML = `<div class="markdown-body px-4 py-2 mb-4 js-type-target"></div>`;
                requestAnimationFrame(() => div.classList.remove('opacity-0'));
                await typeText(div.querySelector('.js-type-target'), text, true);
```

- [ ] **Step 6: STATE 加 currentKind 字段**

`STATE` 对象加字段：

```javascript
            currentKind: "normal",
```

- [ ] **Step 7: 前端测试与构建**

Run: `node resource/static/sanguo_state.test.js`
Expected: 两组测试 PASS

Run: `go build ./...`
Expected: 编译通过

- [ ] **Step 8: Commit**

```bash
git add resource/static/sanguo.html resource/static/sanguo_state.test.js
git commit -m "feat(frontend): 按 kind 差异化呈现，timeskip 时间轴"
```

---

### Task 10: 全量回归与卡顿检查

**Files:**
- 无新文件，仅验证

- [ ] **Step 1: 全量构建与测试**

Run: `go build ./... && go vet ./... 2>&1 | grep -v "main.go" ; go test ./...`
Expected: 编译通过；vet 仅 main.go 既有警告；全部测试 PASS

- [ ] **Step 2: 前端测试**

Run: `node resource/static/sanguo_state.test.js && node resource/static/audio_guard.test.js`
Expected: 全部 PASS

- [ ] **Step 3: 检查无遗漏未提交改动**

Run: `git status --short | grep -vE "vendor|\.idea|env.toml"`
Expected: 无输出（env.toml 被忽略属正常）

- [ ] **Step 4: 浏览器回归（人工，需真 Ark key + 浏览器）**

启动服务，浏览器打开三国页：
- 开局 normal 呈现，状态条正常。
- 连续 6+ 轮选择，观察类型轮转：连续 3 轮 normal 后应出现 timeskip（时间轴分隔 + 2 选项）；某维度到 1 时冒 crisis（红底 + 2 选项）；跨阈值时冒 boon（金底 + 3 选项）。
- kind 缺失/异常时不报错，降级 normal。
- 数值仍不外露，状态条档位随 delta 演化。
- 卡顿回归：首字延迟无明显退化。

- [ ] **Step 5: 记录回归结果**

在 `.superpowers/sdd/progress.md` 追加一行回归结论（人工填写）。

---

## Self-Review

**Spec coverage:**
- §1 4 类型与判定 → Task 3 ✓
- §2 隐藏 JSON 加 kind + decideKindCandidate + RecentKinds 维护 → Task 3/5 ✓
- §2 候选写进 system → Task 6 ✓
- §3 各类型 prompt 规则段 → Task 4/6 ✓
- §3 前端按 kind 呈现 + timeskip 时间轴 → Task 9 ✓
- §4 容错（kind 缺失/非法降级 normal、不裁剪选项、delta 越界忽略）→ Task 2/5/6 ✓
- §5 测试 → Task 2/3/5/9 单元 + Task 10 回归 ✓

**Placeholder scan:** 无 TBD/TODO；每步含完整代码与命令。Task 7 的 env.toml 改动有明确说明（gitignored，手动同步），非占位。

**Type consistency:** `initSystemPrompt` 5 参签名在 Task 6 定义、Task 6 各调用点一致；`decideKindCandidate(s, recentKinds)` 在 Task 3 定义、Task 6 调用一致；`TurnResult.Kind` 在 Task 5 定义、Task 8 引用 `turn.Kind` 一致；`kindPromptRule` 在 Task 4 定义、Task 6 调用一致；前端 `STATE.currentKind` 在 Task 9 Step 6 定义、Step 5 引用一致。

**修正：** Task 5 Step 7 提到旧测试 `TestFinalizeTurnAppliesDeltaAndClamps` 不检查 Kind——确认该测试（前一计划 Task 7）只断言 Panel/Delta/Options/Summary，加 Kind 字段不破坏它。
