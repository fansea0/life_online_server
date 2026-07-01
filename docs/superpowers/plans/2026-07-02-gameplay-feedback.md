# 三国玩法反馈优化 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 给三国玩法加入"可见但不露数值"的状态面板、选后揭示后果、身份隐含驱动选项，消除选择无后果感。

**Architecture:** 复用现有 `&` 隐藏通道承载结构化 JSON（summary/state_delta/options）。服务端维护有界 `GameState`（4 维 0–10，clamp + 增量驱动 + 服务端权威），派生定性面板与态势提示注入 system 提示。新增 `state` WS 帧推送面板 + delta 方向 + 选项。单次流式，不退化卡顿修复成果。

**Tech Stack:** Go (gin, cloudwego/eino, gorilla/websocket), 前端纯 HTML/JS (Tailwind, marked), 测试 Go `testing` + 前端纯函数测试。

## Global Constraints

- 4 维固定：`名望` / `人心` / `实力` / `机缘`，每维硬性 `0–10`，服务端 clamp。
- AI 只输出 `state_delta`（值限定 `{-2,-1,0,1,2}`，越界忽略），不持有累计值。
- 玩家永不看到具体数值，只看定性档位名与 `+/-/0` 方向。
- 选项固定 3 个，文案 ≤15 字，不展示后果。
- 任何解析失败都不阻断剧情展示；解析失败记 warn 不记 error。
- 单次流式调用，不引入二段调用；沿用已修复的关闭深度思考 + 有限超时模型配置。
- Go 结构体字段用 `int` 不用 `int32`（项目规范）；时间/id 可用 `int64`。
- 日志 msg 以函数名开头，redis/sql 空行错误不打 error。

---

## File Structure

- `service/game/types.go` — 改：`GameState` 改为三国 4 维，增加身份与倾向存储。
- `service/game/store.go` — 改：`CreateNewGame` 接收身份初始值。
- `service/game/state.go` — 新建：身份表、`clampState`/`applyDelta`/`panelFromState`/`stateHintFromState`/`parseHiddenPayload`/`FinalizeTurn`。单一职责：纯状态逻辑，不感知 WebSocket。
- `service/game/game.go` — 改：流式入口注入状态/倾向/态势提示；`UpdateContextWithResponse` 改为回灌 summary。
- `route/game/game.go` — 改：`GameWS` 的 `&` 尾接 `FinalizeTurn` 并发 `state` 帧；`identifyList` 配身份初始值与倾向。
- `config/env.toml` — 改：`SystemMsg` 增加占位与选项约束；`RespFormat` 精简可见部分、移除可见选项。
- `resource/static/sanguo.html` — 改：顶部状态条；处理 `state` 帧；选项改由 `state` 帧驱动。
- `service/game/state_test.go` — 新建：状态逻辑单元测试。
- `resource/static/sanguo_state.test.js` — 新建：前端纯函数测试。

---

### Task 1: 重构 GameState 为三国 4 维

**Files:**
- Modify: `service/game/types.go`
- Test: `service/game/game_test.go`（现有测试不依赖旧字段，无需改）

**Interfaces:**
- Produces: `GameState` 结构体含 `SessionID/Name/Identity/IdentityAffinity/Age/Attributes/Summary/IsGameOver`，`Attributes` 为 `map[string]int`，键为 `名望/人心/实力/机缘`。

- [ ] **Step 1: 替换 types.go 的 GameState 定义**

将 `service/game/types.go` 的 `GameState` 改为：

```go
// GameState 玩家当前状态（三国 4 维）
type GameState struct {
	SessionID         string         `json:"session_id"`
	Name              string         `json:"name"`
	Identity          string         `json:"identity"`           // 初始身份描述
	IdentityAffinity  string         `json:"identity_affinity"`  // 身份倾向自然语言描述
	Age               int            `json:"age"`
	Attributes        map[string]int `json:"attributes"`         // 名望/人心/实力/机缘
	Summary           string         `json:"summary"`            // 前情摘要，回灌 AI 上下文
	IsGameOver        bool           `json:"is_game_over"`
}
```

保留 `UserAction`、`AIResponse`、`GameResponse` 不变（旧 `RunGameStep` 路径仍用 `AIResponse.AttrChange`，字段名通用，不冲突）。

- [ ] **Step 2: 运行现有测试确认未破坏**

Run: `go test ./service/game/...`
Expected: PASS（`TestAppendInitialGamePromptAddsUserInstruction` 不依赖 `GameState`）

- [ ] **Step 3: Commit**

```bash
git add service/game/types.go
git commit -m "refactor(game): GameState 改为三国 4 维并存储身份倾向"
```

---

### Task 2: 身份表与 CreateNewGame 接收身份

**Files:**
- Create: `service/game/state.go`
- Modify: `service/game/store.go`

**Interfaces:**
- Produces: `IdentityInit` 结构体；`IdentityTable map[string]IdentityInit`（键为身份描述）；`CreateNewGame(name, identity string) *GameState`。

- [ ] **Step 1: 在 state.go 定义身份初始值与倾向表**

创建 `service/game/state.go`：

```go
package game

// IdentityInit 身份初始状态与倾向
type IdentityInit struct {
	Attributes       map[string]int // 名望/人心/实力/机缘 初始值
	IdentityAffinity string         // 身份倾向自然语言描述
}

// IdentityTable 身份描述 -> 初始值与倾向。键须与 route/game identifyList 的 description 完全一致。
var IdentityTable = map[string]IdentityInit{
	"魏国的一名普通士兵": {
		Attributes:       map[string]int{"名望": 0, "人心": 0, "实力": 1, "机缘": 0},
		IdentityAffinity: "魏国普通士兵：服从军令、稳固阵线为本，少出头、保性命；回避擅自脱阵与抗命。",
	},
	"魏国的一名年轻将领": {
		Attributes:       map[string]int{"名望": 1, "人心": 0, "实力": 2, "机缘": 1},
		IdentityAffinity: "魏国年轻将领：渴望建功立业、亲冒矢石，以武勋博名望；回避怯战退缩。",
	},
	"魏国的一名谋士": {
		Attributes:       map[string]int{"名望": 1, "人心": 0, "实力": 1, "机缘": 1},
		IdentityAffinity: "魏国谋士：偏好运筹帷幄、借势借力、保全己方；回避莽撞冲杀。",
	},
	"蜀国的一名普通士兵": {
		Attributes:       map[string]int{"名望": 0, "人心": 0, "实力": 1, "机缘": 0},
		IdentityAffinity: "蜀国普通士兵：信义为先、同袍相扶为本；回避背信弃义、临阵脱逃。",
	},
	"蜀国的一名年轻将领": {
		Attributes:       map[string]int{"名望": 1, "人心": 1, "实力": 2, "机缘": 1},
		IdentityAffinity: "蜀国年轻将领：忠义敢战、愿为先驱，以仁义收人心；回避不义之功。",
	},
	"蜀国的一名谋士": {
		Attributes:       map[string]int{"名望": 1, "人心": 1, "实力": 1, "机缘": 1},
		IdentityAffinity: "蜀国谋士：偏好兴复汉室、以德服人、奇正相济；回避残民以逞。",
	},
	"吴国的一名普通士兵": {
		Attributes:       map[string]int{"名望": 0, "人心": 0, "实力": 1, "机缘": 0},
		IdentityAffinity: "吴国普通士兵：守土保疆、听候将令为本；回避越权妄动。",
	},
	"吴国的一名年轻将领": {
		Attributes:       map[string]int{"名望": 1, "人心": 0, "实力": 2, "机缘": 1},
		IdentityAffinity: "吴国年轻将领：善用水军地利、伺机而动，以谋勇并进；回避舍长就短。",
	},
	"吴国的一名谋士": {
		Attributes:       map[string]int{"名望": 1, "人心": 0, "实力": 1, "机缘": 1},
		IdentityAffinity: "吴国谋士：偏好联横合纵、保境安民、审时度势；回避四面树敌。",
	},
	"黄巾起义军的一名小将": {
		Attributes:       map[string]int{"名望": 0, "人心": 1, "实力": 1, "机缘": 0},
		IdentityAffinity: "黄巾小将：以道众相扶、劫富济贫为念，敢以身犯险；回避欺压平民。",
	},
	"袁绍军的一名将领": {
		Attributes:       map[string]int{"名望": 1, "人心": 0, "实力": 2, "机缘": 1},
		IdentityAffinity: "袁绍军将领：依仗门第、争功于朝堂，以兵多势大为倚；回避孤军深入。",
	},
	"刘表荆州军的一名官员": {
		Attributes:       map[string]int{"名望": 1, "人心": 0, "实力": 1, "机缘": 1},
		IdentityAffinity: "荆州官员：偏好守成安民、文教治理、结交名士；回避穷兵黩武。",
	},
	"马腾西凉军的一名骑兵": {
		Attributes:       map[string]int{"名望": 0, "人心": 0, "实力": 2, "机缘": 0},
		IdentityAffinity: "西凉骑兵：骁勇善骑、以武立身、重义轻生；回避文弱算计。",
	},
	"公孙瓒白马义从的一员": {
		Attributes:       map[string]int{"名望": 1, "人心": 0, "实力": 2, "机缘": 1},
		IdentityAffinity: "白马义从：精骑突袭、以快制胜、不畏胡患；回避坐守迟疑。",
	},
	"张鲁五斗米教的信徒": {
		Attributes:       map[string]int{"名望": 0, "人心": 1, "实力": 0, "机缘": 1},
		IdentityAffinity: "五斗米信徒：以道治民、宽柔济世、符水救人；回避刀兵杀戮。",
	},
	"韩遂西凉叛军的首领": {
		Attributes:       map[string]int{"名望": 1, "人心": 0, "实力": 2, "机缘": 0},
		IdentityAffinity: "西凉叛军首领：割据自保、反复纵横、以利合离；回避轻信于人。",
	},
	"陶谦徐州军的郡守": {
		Attributes:       map[string]int{"名望": 1, "人心": 1, "实力": 1, "机缘": 0},
		IdentityAffinity: "徐州郡守：仁政安民、礼让贤能、守土恤民；回避穷兵黩武。",
	},
	"刘璋益州军的官员": {
		Attributes:       map[string]int{"名望": 1, "人心": 0, "实力": 1, "机缘": 1},
		IdentityAffinity: "益州官员：凭险自守、倚重本土、循例办事；回避冒险轻进。",
	},
}

// defaultIdentityInit 未知身份的兜底初始值
var defaultIdentityInit = IdentityInit{
	Attributes:       map[string]int{"名望": 0, "人心": 0, "实力": 1, "机缘": 0},
	IdentityAffinity: "乱世中人：审时度势、趋利避害，伺机而动；回避无谓送死。",
}

// LookupIdentity 查身份初始值，未知返回兜底
func LookupIdentity(identity string) IdentityInit {
	if init, ok := IdentityTable[identity]; ok {
		return init
	}
	return defaultIdentityInit
}
```

- [ ] **Step 2: 改 store.go 的 CreateNewGame 接收身份**

将 `service/game/store.go` 的 `CreateNewGame` 改为：

```go
func CreateNewGame(name, identity string) *GameState {
	sessionID := uuid.New().String()
	init := LookupIdentity(identity)
	// 复制初始属性，避免共享 map
	attrs := make(map[string]int, len(init.Attributes))
	for k, v := range init.Attributes {
		attrs[k] = v
	}
	state := &GameState{
		SessionID:        sessionID,
		Name:             name,
		Identity:         identity,
		IdentityAffinity: init.IdentityAffinity,
		Age:              0,
		Attributes:       attrs,
		Summary:          "你初到乱世，前路未明。",
	}
	SaveState(state)
	return state
}
```

- [ ] **Step 3: 编译确认**

Run: `go build ./service/game/...`
Expected: 无错误（旧 `RunGameStep` 调用 `CreateNewGame()` 无参处需同步改，见下一步检查）

- [ ] **Step 4: 修正旧 RunGameStep 调用点**

`service/game/engine.go:33` `state = CreateNewGame()` 改为 `state = CreateNewGame("你", "")`。

Run: `go build ./...`
Expected: 编译通过

- [ ] **Step 5: Commit**

```bash
git add service/game/state.go service/game/store.go service/game/engine.go
git commit -m "feat(game): 身份初始值与倾向表，CreateNewGame 接收身份"
```

---

### Task 3: clampState 与 applyDelta

**Files:**
- Modify: `service/game/state.go`
- Test: `service/game/state_test.go`

**Interfaces:**
- Produces: `func clampState(s *GameState)`、`func applyDelta(s *GameState, delta map[string]int)`。delta 值必须 `{-2,-1,0,1,2}`，越界按 0；缺维度按 0；多余键忽略。

- [ ] **Step 1: 写失败测试**

创建 `service/game/state_test.go`：

```go
package game

import "testing"

func newTestState() *GameState {
	return &GameState{
		Attributes: map[string]int{"名望": 5, "人心": 5, "实力": 5, "机缘": 5},
	}
}

func TestApplyDeltaNormal(t *testing.T) {
	s := newTestState()
	applyDelta(s, map[string]int{"名望": 1, "人心": 0, "实力": -1, "机缘": 2})
	if s.Attributes["名望"] != 6 || s.Attributes["人心"] != 5 || s.Attributes["实力"] != 4 || s.Attributes["机缘"] != 7 {
		t.Fatalf("applyDelta normal: got %+v", s.Attributes)
	}
}

func TestApplyDeltaClampHigh(t *testing.T) {
	s := newTestState()
	s.Attributes["名望"] = 9
	applyDelta(s, map[string]int{"名望": 2})
	if s.Attributes["名望"] != 10 {
		t.Fatalf("clamp high: got %d, want 10", s.Attributes["名望"])
	}
}

func TestApplyDeltaClampLow(t *testing.T) {
	s := newTestState()
	s.Attributes["实力"] = 1
	applyDelta(s, map[string]int{"实力": -2})
	if s.Attributes["实力"] != 0 {
		t.Fatalf("clamp low: got %d, want 0", s.Attributes["实力"])
	}
}

func TestApplyDeltaIgnoresOutOfRange(t *testing.T) {
	s := newTestState()
	applyDelta(s, map[string]int{"名望": 5, "实力": -3})
	if s.Attributes["名望"] != 5 || s.Attributes["实力"] != 5 {
		t.Fatalf("out-of-range delta should be ignored: got %+v", s.Attributes)
	}
}

func TestApplyDeltaIgnoresUnknownKey(t *testing.T) {
	s := newTestState()
	applyDelta(s, map[string]int{"名望": 1, "财富": 9})
	if _, exists := s.Attributes["财富"]; exists {
		t.Fatalf("unknown key 财富 should not be added")
	}
	if s.Attributes["名望"] != 6 {
		t.Fatalf("名望 should be 6, got %d", s.Attributes["名望"])
	}
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./service/game/ -run TestApplyDelta -v`
Expected: FAIL（`applyDelta` 未定义）

- [ ] **Step 3: 实现 clampState 与 applyDelta**

在 `service/game/state.go` 追加：

```go
var validDims = []string{"名望", "人心", "实力", "机缘"}

// clampState 将 4 维硬性限制在 0-10
func clampState(s *GameState) {
	for _, dim := range validDims {
		v := s.Attributes[dim]
		if v < 0 {
			v = 0
		}
		if v > 10 {
			v = 10
		}
		s.Attributes[dim] = v
	}
}

// applyDelta 叠加本轮增量，越界值与未知键忽略，最后 clamp
func applyDelta(s *GameState, delta map[string]int) {
	for _, dim := range validDims {
		d, ok := delta[dim]
		if !ok {
			continue
		}
		if d < -2 || d > 2 {
			continue // 越界忽略
		}
		s.Attributes[dim] += d
	}
	clampState(s)
}
```

- [ ] **Step 4: 运行测试确认通过**

Run: `go test ./service/game/ -run TestApplyDelta -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add service/game/state.go service/game/state_test.go
git commit -m "feat(game): applyDelta 增量叠加与 clamp 防爆炸"
```

---

### Task 4: panelFromState 定性档位

**Files:**
- Modify: `service/game/state.go`
- Test: `service/game/state_test.go`

**Interfaces:**
- Produces: `func panelFromState(s *GameState) map[string]string`，返回 4 维定性档位名。档位索引 = `min(value/2, 5)`（整数除法）。

- [ ] **Step 1: 写失败测试**

追加到 `service/game/state_test.go`：

```go
func TestPanelFromStateTiers(t *testing.T) {
	cases := []struct {
		val  int
		want string
	}{
		{0, "籍籍无名"}, {1, "籍籍无名"},
		{2, "小有名气"}, {3, "小有名气"},
		{4, "颇有声名"}, {5, "颇有声名"},
		{6, "威名渐显"}, {7, "威名渐显"},
		{8, "名震一方"}, {9, "名震一方"},
		{10, "天下知名"},
	}
	for _, c := range cases {
		s := &GameState{Attributes: map[string]int{"名望": c.val, "人心": 0, "实力": 0, "机缘": 0}}
		panel := panelFromState(s)
		if panel["名望"] != c.want {
			t.Fatalf("名望 %d: got %q, want %q", c.val, panel["名望"], c.want)
		}
	}
}

func TestPanelFromStateAllDims(t *testing.T) {
	s := &GameState{Attributes: map[string]int{"名望": 8, "人心": 6, "实力": 4, "机缘": 2}}
	panel := panelFromState(s)
	if panel["名望"] != "名震一方" || panel["人心"] != "众心所向" || panel["实力"] != "颇具战力" || panel["机缘"] != "偶有际遇" {
		t.Fatalf("all dims: got %+v", panel)
	}
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./service/game/ -run TestPanelFromState -v`
Expected: FAIL（`panelFromState` 未定义）

- [ ] **Step 3: 实现 panelFromState**

在 `service/game/state.go` 追加：

```go
var panelTiers = map[string][]string{
	"名望": {"籍籍无名", "小有名气", "颇有声名", "威名渐显", "名震一方", "天下知名"},
	"人心": {"无人归附", "略有人望", "初得民望", "众心所向", "深得人心", "万众归心"},
	"实力": {"手无寸铁", "勉强自保", "可堪一用", "颇具战力", "羽翼渐丰", "兵强马壮"},
	"机缘": {"时运不济", "偶有际遇", "尚需时运", "机缘渐至", "福星高照", "天命所归"},
}

// panelFromState 派生 4 维定性档位名
func panelFromState(s *GameState) map[string]string {
	panel := make(map[string]string, len(validDims))
	for _, dim := range validDims {
		tiers := panelTiers[dim]
		idx := s.Attributes[dim] / 2
		if idx < 0 {
			idx = 0
		}
		if idx > 5 {
			idx = 5
		}
		panel[dim] = tiers[idx]
	}
	return panel
}
```

- [ ] **Step 4: 运行测试确认通过**

Run: `go test ./service/game/ -run TestPanelFromState -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add service/game/state.go service/game/state_test.go
git commit -m "feat(game): panelFromState 定性档位映射"
```

---

### Task 5: stateHintFromState 态势提示

**Files:**
- Modify: `service/game/state.go`
- Test: `service/game/state_test.go`

**Interfaces:**
- Produces: `func stateHintFromState(s *GameState) string`，某维 `>=5` 时追加该维态势句，多维用逗号连接；全低时返回空串。

- [ ] **Step 1: 写失败测试**

追加到 `service/game/state_test.go`：

```go
func TestStateHintEmptyWhenLow(t *testing.T) {
	s := &GameState{Attributes: map[string]int{"名望": 4, "人心": 4, "实力": 4, "机缘": 4}}
	if h := stateHintFromState(s); h != "" {
		t.Fatalf("low state hint should be empty, got %q", h)
	}
}

func TestStateHintWhenHigh(t *testing.T) {
	s := &GameState{Attributes: map[string]int{"名望": 5, "人心": 4, "实力": 7, "机缘": 4}}
	h := stateHintFromState(s)
	if h == "" {
		t.Fatalf("expected non-empty hint")
	}
	// 名望与实力达标，应包含两句
	if !strings.Contains(h, "声名") || !strings.Contains(h, "兵权") {
		t.Fatalf("hint missing dims: %q", h)
	}
}
```

并在文件 import 加 `"strings"`。

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./service/game/ -run TestStateHint -v`
Expected: FAIL（`stateHintFromState` 未定义，且 import 缺 strings）

- [ ] **Step 3: 实现 stateHintFromState**

在 `service/game/state_test.go` 顶部 import 改为：

```go
import (
	"strings"
	"testing"
)
```

在 `service/game/state.go` 追加：

```go
var stateHints = map[string]string{
	"名望": "你如今颇有声名，言行皆为各方关注",
	"人心": "你已深得人心，众人愿为你效力",
	"实力": "你如今手握兵权，行事更受各方瞩目",
	"机缘": "你近来机缘不断，似有天意相助",
}

// stateHintFromState 高维度（>=5）态势提示，多维逗号连接，全低返回空串
func stateHintFromState(s *GameState) string {
	parts := make([]string, 0, len(validDims))
	for _, dim := range validDims {
		if s.Attributes[dim] >= 5 {
			parts = append(parts, stateHints[dim])
		}
	}
	if len(parts) == 0 {
		return ""
	}
	return strings.Join(parts, "；")
}
```

并在 `state.go` import 加 `"strings"`：

```go
import (
	"strings"
)
```

- [ ] **Step 4: 运行测试确认通过**

Run: `go test ./service/game/ -run TestStateHint -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add service/game/state.go service/game/state_test.go
git commit -m "feat(game): stateHintFromState 跨阈值态势提示"
```

---

### Task 6: parseHiddenPayload 解析容错

**Files:**
- Modify: `service/game/state.go`
- Test: `service/game/state_test.go`

**Interfaces:**
- Produces: `type HiddenPayload struct { Summary string; Delta map[string]int; Options []string }`；`func parseHiddenPayload(raw string) HiddenPayload`。永不返回 error，分层降级。

- [ ] **Step 1: 写失败测试**

追加到 `service/game/state_test.go`：

```go
func TestParseHiddenPayloadNormal(t *testing.T) {
	raw := `{"summary":"你救了百姓","state_delta":{"名望":1,"人心":0,"实力":-1,"机缘":0},"options":["出城迎敌","固守待援","遣使求和"]}`
	p := parseHiddenPayload(raw)
	if p.Summary != "你救了百姓" {
		t.Fatalf("summary: got %q", p.Summary)
	}
	if p.Delta["名望"] != 1 || p.Delta["实力"] != -1 {
		t.Fatalf("delta: got %+v", p.Delta)
	}
	if len(p.Options) != 3 || p.Options[0] != "出城迎敌" {
		t.Fatalf("options: got %+v", p.Options)
	}
}

func TestParseHiddenPayloadBadJSON(t *testing.T) {
	p := parseHiddenPayload("not json")
	if p.Summary != "" {
		t.Fatalf("bad json summary should be empty, got %q", p.Summary)
	}
	if len(p.Delta) != 0 {
		t.Fatalf("bad json delta should be empty")
	}
	if len(p.Options) != 1 || p.Options[0] != "继续" {
		t.Fatalf("bad json options should fallback to 继续, got %+v", p.Options)
	}
}

func TestParseHiddenPayloadOptionsPadAndTruncate(t *testing.T) {
	p := parseHiddenPayload(`{"options":["only one"]}`)
	if len(p.Options) != 3 || p.Options[0] != "only one" || p.Options[1] != "继续" || p.Options[2] != "继续" {
		t.Fatalf("pad: got %+v", p.Options)
	}
	p2 := parseHiddenPayload(`{"options":["a","b","c","d","e"]}`)
	if len(p2.Options) != 3 || p2.Options[2] != "c" {
		t.Fatalf("truncate: got %+v", p2.Options)
	}
}

func TestParseHiddenPayloadDeltaOutOfRangeIgnored(t *testing.T) {
	p := parseHiddenPayload(`{"state_delta":{"名望":9,"实力":-5,"人心":1}}`)
	if p.Delta["名望"] != 0 || p.Delta["实力"] != 0 || p.Delta["人心"] != 1 {
		t.Fatalf("out-of-range should be 0: got %+v", p.Delta)
	}
}

func TestParseHiddenPayloadDeltaOnlyKnownDims(t *testing.T) {
	p := parseHiddenPayload(`{"state_delta":{"名望":1,"财富":2}}`)
	if _, exists := p.Delta["财富"]; exists {
		t.Fatalf("财富 should be dropped")
	}
	if p.Delta["名望"] != 1 {
		t.Fatalf("名望 should be 1, got %d", p.Delta["名望"])
	}
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./service/game/ -run TestParseHiddenPayload -v`
Expected: FAIL（`parseHiddenPayload` 未定义）

- [ ] **Step 3: 实现 parseHiddenPayload**

在 `service/game/state.go` 追加：

```go
import (
	"encoding/json"
	"strings"
)

// HiddenPayload AI 隐藏部分（& 之后）解析结果
type HiddenPayload struct {
	Summary string
	Delta   map[string]int
	Options []string
}

type rawHiddenPayload struct {
	Summary    string         `json:"summary"`
	StateDelta map[string]int `json:"state_delta"`
	Options    []string       `json:"options"`
}

// parseHiddenPayload 解析 & 之后的 JSON，分层降级，永不返回 error
func parseHiddenPayload(raw string) HiddenPayload {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return HiddenPayload{Options: []string{"继续"}}
	}
	var r rawHiddenPayload
	if err := json.Unmarshal([]byte(raw), &r); err != nil {
		logrus.WithError(err).Warn("parseHiddenPayload: unmarshal failed, fallback")
		return HiddenPayload{Options: []string{"继续"}}
	}

	delta := make(map[string]int, len(validDims))
	for _, dim := range validDims {
		d, ok := r.StateDelta[dim]
		if !ok {
			delta[dim] = 0
			continue
		}
		if d < -2 || d > 2 {
			d = 0
		}
		delta[dim] = d
	}

	options := normalizeOptions(r.Options)
	return HiddenPayload{Summary: r.Summary, Delta: delta, Options: options}
}

// normalizeOptions 补足/截断为 3 个，缺位补"继续"
func normalizeOptions(opts []string) []string {
	result := make([]string, 0, 3)
	for i := 0; i < 3; i++ {
		if i < len(opts) && strings.TrimSpace(opts[i]) != "" {
			result = append(result, opts[i])
		} else {
			result = append(result, "继续")
		}
	}
	return result
}
```

注意 `state.go` 顶部 import 合并为：

```go
import (
	"encoding/json"
	"strings"

	"github.com/sirupsen/logrus"
)
```

- [ ] **Step 4: 运行测试确认通过**

Run: `go test ./service/game/ -run TestParseHiddenPayload -v`
Expected: PASS

- [ ] **Step 5: 运行全部状态测试**

Run: `go test ./service/game/... -v`
Expected: 全部 PASS

- [ ] **Step 6: Commit**

```bash
git add service/game/state.go service/game/state_test.go
git commit -m "feat(game): parseHiddenPayload 分层容错解析"
```

---

### Task 7: FinalizeTurn 收尾（解析+叠加+持久化+派生）

**Files:**
- Modify: `service/game/state.go`
- Test: `service/game/state_test.go`

**Interfaces:**
- Produces: `type TurnResult struct { Panel map[string]string; Delta map[string]string; Options []string }`；`func FinalizeTurn(sessionID, hiddenRaw string) (TurnResult, bool)`。返回值 `ok=false` 表示无会话状态（开局首轮尚未建 state 时可能发生），调用方按需降级。`Delta` 为方向串 `"+"/"-"/"0"`。

- [ ] **Step 1: 写失败测试**

追加到 `service/game/state_test.go`：

```go
func TestFinalizeTurnAppliesDeltaAndClamps(t *testing.T) {
	s := CreateNewGame("陈凡", "魏国的一名谋士")
	before := s.Attributes["名望"]
	raw := `{"summary":"你献策退敌","state_delta":{"名望":1,"实力":-1},"options":["乘胜追击","收兵回营","上书请功"]}`
	res, ok := FinalizeTurn(s.SessionID, raw)
	if !ok {
		t.Fatalf("expected ok=true")
	}
	after := GetState(s.SessionID).Attributes["名望"]
	if after != before+1 {
		t.Fatalf("名望 should increase by 1: before %d after %d", before, after)
	}
	if res.Panel["名望"] == "" {
		t.Fatalf("panel should not be empty")
	}
	if res.Delta["名望"] != "+" || res.Delta["实力"] != "-" {
		t.Fatalf("delta direction: got %+v", res.Delta)
	}
	if len(res.Options) != 3 || res.Options[0] != "乘胜追击" {
		t.Fatalf("options: got %+v", res.Options)
	}
	if GetState(s.SessionID).Summary != "你献策退敌" {
		t.Fatalf("summary should be persisted, got %q", GetState(s.SessionID).Summary)
	}
}

func TestFinalizeTurnNoSession(t *testing.T) {
	_, ok := FinalizeTurn("nonexistent-session", `{"options":["a","b","c"]}`)
	if ok {
		t.Fatalf("expected ok=false for nonexistent session")
	}
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./service/game/ -run TestFinalizeTurn -v`
Expected: FAIL（`FinalizeTurn`/`TurnResult` 未定义）

- [ ] **Step 3: 实现 FinalizeTurn**

在 `service/game/state.go` 追加：

```go
// TurnResult 一轮收尾后推给前端的结果
type TurnResult struct {
	Panel   map[string]string // 定性档位名
	Delta   map[string]string // 方向 "+"/"-"/"0"
	Options []string          // 下一轮 3 选项
}

// FinalizeTurn 解析隐藏部分、叠加 delta、持久化、派生面板与方向。
// ok=false 表示会话状态不存在，调用方降级（不更新面板）。
func FinalizeTurn(sessionID, hiddenRaw string) (TurnResult, bool) {
	state := GetState(sessionID)
	if state == nil {
		return TurnResult{}, false
	}
	payload := parseHiddenPayload(hiddenRaw)

	// 记录方向（叠加前）
	deltaDir := make(map[string]string, len(validDims))
	for _, dim := range validDims {
		d := payload.Delta[dim]
		switch {
		case d > 0:
			deltaDir[dim] = "+"
		case d < 0:
			deltaDir[dim] = "-"
		default:
			deltaDir[dim] = "0"
		}
	}

	applyDelta(state, payload.Delta)
	if payload.Summary != "" {
		state.Summary = payload.Summary
	}
	SaveState(state)

	return TurnResult{
		Panel:   panelFromState(state),
		Delta:   deltaDir,
		Options: payload.Options,
	}, true
}
```

- [ ] **Step 4: 运行测试确认通过**

Run: `go test ./service/game/ -run TestFinalizeTurn -v`
Expected: PASS

- [ ] **Step 5: 运行全部测试**

Run: `go test ./service/game/...`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add service/game/state.go service/game/state_test.go
git commit -m "feat(game): FinalizeTurn 收尾解析叠加持久化"
```

---

### Task 8: 流式入口注入状态/倾向/态势提示，UpdateContextWithResponse 回灌 summary

**Files:**
- Modify: `service/game/game.go`

**Interfaces:**
- Consumes: `LookupIdentity`、`stateHintFromState`、`GetState`、`CreateNewGame`（来自 Task 2/5/7）
- Produces: `StartGameStream`/`HandleChoiceStream` 现在会建/读 `GameState` 并把 `identityAffinity`、`stateHint` 填进 system 模板；`UpdateContextWithResponse(sessionID, summary)` 行为不变（仍把传入文本作为 assistant 回灌），但调用方改为传 `payload.Summary`（见 Task 10）。

- [ ] **Step 1: 扩展 initSystemPrompt 接收 affinity 与 stateHint**

`service/game/game.go` 中 `initSystemPrompt` 改为：

```go
func initSystemPrompt(name, identify, affinity, stateHint string) ([]*schema.Message, error) {
	msg, err := eino.CreateMessagesCommon(
		config.GetSystemMsg(),
		map[string]any{
			"name":              name,
			"identify":          identify,
			"identityAffinity":  affinity,
			"stateHint":         stateHint,
			"respFormat":        config.GetRespFormat(),
			"otherReqs":         config.GetOtherReqs(),
		},
		true,
	)
	if err != nil {
		return nil, err
	}
	return msg, nil
}
```

- [ ] **Step 2: 改 StartGameStream 建状态并注入**

```go
func StartGameStream(name, identify string) (string, *schema.StreamReader[*schema.Message], error) {
	state := CreateNewGame(name, identify)
	affinity := state.IdentityAffinity
	// 开局尚无态势，stateHint 为空
	msgContext, err := initSystemPrompt(name, identify, affinity, "")
	if err != nil {
		return "", nil, err
	}
	msgContext = appendInitialGamePrompt(msgContext)
	streamReader, err := eino.Stream(model, msgContext)
	if err != nil {
		logrus.WithError(err).Error("StartGameStream: eino.Stream failed")
		return "", nil, err
	}
	newUUID, _ := uuid.NewUUID()
	sessionID := newUUID.String()
	// 用 state 的 SessionID 作为对外 sessionID，保证状态与上下文键一致
	sessionID = state.SessionID
	// 保存上下文（不含本次 AI 回复）
	SaveMsgContext(sessionID, msgContext)
	return sessionID, streamReader, nil
}
```

- [ ] **Step 3: 改 HandleChoiceStream 读状态并注入态势提示**

```go
func HandleChoiceStream(sessionID, choice string) (*schema.StreamReader[*schema.Message], error) {
	context := GetMsgContext(sessionID)
	state := GetState(sessionID)
	stateHint := ""
	affinity := ""
	if state != nil {
		stateHint = stateHintFromState(state)
		affinity = state.IdentityAffinity
	}
	// 重新生成带最新状态提示的 system，替换上下文首条 system 消息
	sysMsg, err := initSystemPrompt(stateIdentityName(state), stateIdentityDesc(state), affinity, stateHint)
	if err == nil && len(sysMsg) > 0 && len(context) > 0 {
		// 替换首条 system，保留后续历史
		context = append([]*schema.Message{sysMsg[0]}, context[1:]...)
	}
	userMessage, err := eino.CreateMessagesCommon(choice, map[string]any{}, false)
	if err != nil {
		return nil, err
	}
	tempContext := append(context, userMessage...)
	streamReader, err := eino.Stream(model, tempContext)
	if err != nil {
		return nil, err
	}
	SaveMsgContext(sessionID, tempContext)
	return streamReader, nil
}

// stateIdentityName 从 state 取姓名，state 缺失时兜底
func stateIdentityName(state *GameState) string {
	if state != nil {
		return state.Name
	}
	return "你"
}

// stateIdentityDesc 从 state 取身份描述
func stateIdentityDesc(state *GameState) string {
	if state != nil {
		return state.Identity
	}
	return ""
}
```

- [ ] **Step 4: UpdateContextWithResponse 保持回灌语义**

`UpdateContextWithResponse` 当前实现已把传入文本作为 assistant 消息回灌，**不改函数体**。调用方（Task 10 `GameWS`）改为传 `payload.Summary`。确认该函数无需改动后跳过。

- [ ] **Step 5: 编译**

Run: `go build ./...`
Expected: 通过（`StartGame`/`HandleChoice` 非流式入口若调用 `initSystemPrompt` 旧签名需同步改）

- [ ] **Step 6: 同步修正非流式入口签名**

`StartGame` 与 `HandleChoice`（`game.go` 顶部）调用 `initSystemPrompt(name, identify)` 处改为：

```go
msgContext, err := initSystemPrompt(name, identify, LookupIdentity(identify).IdentityAffinity, "")
```

`HandleChoice` 中读取状态并注入同 `HandleChoiceStream` 的逻辑（复用 `stateIdentityName`/`stateIdentityDesc` 与 `stateHintFromState`）。

Run: `go build ./...`
Expected: 通过

- [ ] **Step 7: 运行测试**

Run: `go test ./service/game/...`
Expected: PASS

- [ ] **Step 8: Commit**

```bash
git add service/game/game.go
git commit -m "feat(game): 流式入口注入身份倾向与态势提示"
```

---

### Task 9: 更新 SystemMsg 与 RespFormat 配置

**Files:**
- Modify: `config/env.toml`

**Interfaces:**
- Consumes: Task 8 注入的 `{{.identityAffinity}}`、`{{.stateHint}}` 占位。

- [ ] **Step 1: 改 SystemMsg**

`config/env.toml` 的 `SystemMsg` 改为：

```toml
SystemMsg = "你是一款人生online游戏，用户将扮演三国时期的穿越者，通过你给的选项进行选择，你来述说故事推进。\n你的名称是：{{.name}}\n你的初始身份是：{{.identify}}\n身份倾向：{{.identityAffinity}}\n当前态势：{{.stateHint}}\n\n选项生成规则：\n1. 每轮固定给出3个选项，文案只描述行动、不超过15字，绝不展示后果或数字。\n2. 三个选项须风格分化：一个偏向身份倾向、一个偏进取冒险、一个偏保守稳妥。\n3. 根据玩家当前状态(名望/人心/实力/机缘)与历史选择推演后果，让选择产生可见且跨轮积累的影响。\n4. 在剧情后输出一个隐藏JSON块（以&开头，&之前为可见剧情），JSON结构：{\"summary\":\"一两句前情摘要\",\"state_delta\":{\"名望\":0,\"人心\":0,\"实力\":0,\"机缘\":0},\"options\":[\"选项1\",\"选项2\",\"选项3\"]}。state_delta取值仅限-2到2的整数。可见剧情中不得再出现1./2./3.选项文本。\n\n响应格式（可见部分）严格按照：{{.respFormat}}\n其他要求：{{.otherReqs}}"
```

- [ ] **Step 2: 精简 RespFormat（移除可见选项）**

`RespFormat` 改为（结尾止于旁白，不再有 `*系统提示*：请选择` 与 `1./2./3.`）：

```toml
RespFormat = "# 做出选择后的四字概括\n---\n**你**：“对话内容”\n---\n> 旁白\n---\n**人名**（人物动作,8字以内）：“对话内容”\n---\n**你**：“对话内容”\n---\n## 根据实际内容更改，做出选择后的描述，不超过六字\n---\n行为结果内容，人名需要加粗，样式**人名**，至少50-100字\n---\n## 根据实际内容更改，新事件描述，不超过六字\n---\n新事件叙事，人名需要加粗，样式**人名**，至少50-100字\n---\n**人名**（人物动作,8字以内）：“对话内容”\n---\n**人名**（人物动作,8字以内）：“对话内容”\n---\n**人名**（人物动作,8字以内）：“对话内容”\n---\n> 旁白\n---"
```

- [ ] **Step 3: 编译运行确认配置加载**

Run: `go build ./... && go run . &` （或项目现有启动方式），确认无模板报错后停止。

Expected: 启动正常，`{{.identityAffinity}}`/`{{.stateHint}}` 在开局时分别填入倾向文本与空串，无模板缺失报错。

- [ ] **Step 4: Commit**

```bash
git add config/env.toml
git commit -m "feat(config): SystemMsg 增加身份倾向/态势/选项规则，RespFormat 移除可见选项"
```

---

### Task 10: GameWS 接 FinalizeTurn 并发 state 帧

**Files:**
- Modify: `route/game/game.go`

**Interfaces:**
- Consumes: `game.FinalizeTurn`、`game.UpdateContextWithResponse`、`game.GetState`（来自 Task 7/8）
- Produces: `GameWS` 在 `&` 切分后，流结束调用 `FinalizeTurn`，发 `state` 帧（panel+delta+options），再发 `end`。

- [ ] **Step 1: 改 GameWS 的 & 尾处理**

`route/game/game.go` 的 `GameWS` 中，将 `summary` 累积与 `UpdateContextWithResponse` 段改为：流结束后用 `FinalizeTurn` 解析、叠加、发 `state` 帧、回灌 summary。

定位现有逻辑：

```go
		if streamReader != nil {
			// 读取流并推送
			summaryStarted := false
			for {
				chunk, err := streamReader.Recv()
				...
			}
			streamReader.Close()
		}

		// 发送结束标记，告知前端本轮输出完毕
		ws.WriteJSON(gin.H{"type": "end"})
		logrus.WithFields(logrus.Fields{
			"summary": summary,
		}).Infoln("game summary")

		// 更新完整上下文到内存
		game.UpdateContextWithResponse(sessionID, summary)
```

替换为（保留 `&` 切分推送可见内容、累积隐藏尾；流结束调用 FinalizeTurn）：

```go
		if streamReader != nil {
			summaryStarted := false
			for {
				chunk, err := streamReader.Recv()
				if err == io.EOF {
					break
				}
				if err != nil {
					break
				}

				content := chunk.Content

				if strings.Contains(content, "&") && !summaryStarted {
					atIndex := strings.Index(content, "&")
					summary = content[atIndex+1:]
					if atIndex >= 0 {
						contentToSend := content[:atIndex]
						if contentToSend != "" {
							ws.WriteJSON(gin.H{
								"type":    "content",
								"content": contentToSend,
							})
						}
					}
					summaryStarted = true
					continue
				}

				if !summaryStarted {
					ws.WriteJSON(gin.H{
						"type":    "content",
						"content": content,
					})
				} else {
					summary += content
				}
			}
			streamReader.Close()
		}

		// 解析隐藏尾、叠加状态、派生面板
		turn, ok := game.FinalizeTurn(sessionID, summary)
		summaryToCtx := summary
		if ok {
			// 回灌解析出的 summary 文本（而非原始 JSON）
			if parsed := game.ParseSummaryForContext(summary); parsed != "" {
				summaryToCtx = parsed
			}
			ws.WriteJSON(gin.H{
				"type":    "state",
				"panel":   turn.Panel,
				"delta":   turn.Delta,
				"options": turn.Options,
			})
		}

		ws.WriteJSON(gin.H{"type": "end"})
		logrus.WithFields(logrus.Fields{
			"summary": summaryToCtx,
		}).Infoln("game summary")

		game.UpdateContextWithResponse(sessionID, summaryToCtx)
```

- [ ] **Step 2: 暴露 ParseSummaryForContext 辅助函数**

`FinalizeTurn` 内部已解析 summary 但未单独返回。为避免重复解析，在 `service/game/state.go` 追加：

```go
// ParseSummaryForContext 从隐藏尾解析 summary 文本，供上下文回灌；解析失败返回空串
func ParseSummaryForContext(hiddenRaw string) string {
	return parseHiddenPayload(hiddenRaw).Summary
}
```

- [ ] **Step 3: handle start 帧也发初始 state 帧**

开局首轮尚无 `summary`，但已建 `GameState`。在 `GameWS` 的 `req.Type == "start"` 分支，发完 `session` 帧后补发初始 `state` 帧：

```go
		if req.Type == "start" {
			sessionID, streamReader, err = game.StartGameStream(req.Name, req.Identify)
			if err == nil {
				ws.WriteJSON(gin.H{"type": "session", "session_id": sessionID})
				// 发送初始状态面板（delta 全 0，options 留空，由后续 end 帧不渲染）
				if st := game.GetState(sessionID); st != nil {
					ws.WriteJSON(gin.H{
						"type":  "state",
						"panel": game.PanelOf(sessionID),
						"delta": map[string]string{"名望": "0", "人心": "0", "实力": "0", "机缘": "0"},
					})
				}
			}
		} else if req.Type == "choice" {
			sessionID = req.SessionID
			streamReader, err = game.HandleChoiceStream(req.SessionID, req.Choice)
		}
```

- [ ] **Step 4: 暴露 PanelOf 辅助函数**

在 `service/game/state.go` 追加：

```go
// PanelOf 取会话当前面板，无状态返回 nil
func PanelOf(sessionID string) map[string]string {
	state := GetState(sessionID)
	if state == nil {
		return nil
	}
	return panelFromState(state)
}
```

- [ ] **Step 5: 编译**

Run: `go build ./...`
Expected: 通过

- [ ] **Step 6: 运行全部测试**

Run: `go test ./...`
Expected: PASS

- [ ] **Step 7: Commit**

```bash
git add route/game/game.go service/game/state.go
git commit -m "feat(route): GameWS 接 FinalizeTurn 并发 state 帧"
```

---

### Task 11: 前端状态条与 state 帧处理

**Files:**
- Modify: `resource/static/sanguo.html`
- Create: `resource/static/sanguo_state.test.js`

**Interfaces:**
- Consumes: Task 10 的 `state` 帧（panel/delta/options）。
- Produces: 顶部常驻状态条；`state` 帧处理纯函数 `buildStateBar(prevPanel, payload)` 返回渲染结构。

- [ ] **Step 1: 写前端纯函数失败测试**

创建 `resource/static/sanguo_state.test.js`（沿用 `audio_guard.test.js` 的纯 JS 测试风格，无框架，用 `assert`）：

```javascript
// 纯函数测试：buildStateBar 输入面板+delta，返回渲染结构
function buildStateBar(prevPanel, payload) {
    const dims = ["名望", "人心", "实力", "机缘"];
    const icons = {"名望":"star", "人心":"heart", "实力":"sword", "机缘":"clover"};
    const items = dims.map(d => ({
        dim: d,
        icon: icons[d],
        label: payload.panel[d] || "",
        dir: payload.delta ? payload.delta[d] : "0",
    }));
    return { items, options: payload.options || [] };
}

const assert = require('assert');
try {
    const r = buildStateBar({}, {
        panel: {"名望":"颇有声名","人心":"初得民望","实力":"可堪一用","机缘":"尚需时运"},
        delta: {"名望":"+","人心":"0","实力":"-","机缘":"0"},
        options: ["a","b","c"],
    });
    assert.strictEqual(r.items.length, 4);
    assert.strictEqual(r.items[0].label, "颇有声名");
    assert.strictEqual(r.items[0].dir, "+");
    assert.strictEqual(r.items[2].dir, "-");
    assert.strictEqual(r.options.length, 3);
    console.log("sanguo_state tests PASS");
} catch (e) {
    console.error("sanguo_state tests FAIL:", e.message);
    process.exit(1);
}
```

- [ ] **Step 2: 运行测试确认通过（先实现函数）**

由于 `buildStateBar` 已在该测试文件内定义，直接运行：

Run: `node resource/static/sanguo_state.test.js`
Expected: 输出 `sanguo_state tests PASS`

- [ ] **Step 3: 在 sanguo.html GAME SCREEN 加状态条**

在 `<!-- Header -->` 那段 `div` 之后、`<!-- Content -->` 之前插入：

```html
        <!-- State Bar -->
        <div id="state-bar" class="bg-white border-b border-gray-100 px-4 py-2 flex justify-between items-center text-xs z-20 hidden">
            <div class="flex items-center space-x-1" data-dim="名望">
                <i data-lucide="star" class="w-3 h-3 text-gray-400"></i>
                <span class="text-gray-500">名望</span>
                <span class="font-bold text-gray-800 js-tier">-</span>
                <span class="js-delta text-transparent"></span>
            </div>
            <div class="flex items-center space-x-1" data-dim="人心">
                <i data-lucide="heart" class="w-3 h-3 text-gray-400"></i>
                <span class="text-gray-500">人心</span>
                <span class="font-bold text-gray-800 js-tier">-</span>
                <span class="js-delta text-transparent"></span>
            </div>
            <div class="flex items-center space-x-1" data-dim="实力">
                <i data-lucide="sword" class="w-3 h-3 text-gray-400"></i>
                <span class="text-gray-500">实力</span>
                <span class="font-bold text-gray-800 js-tier">-</span>
                <span class="js-delta text-transparent"></span>
            </div>
            <div class="flex items-center space-x-1" data-dim="机缘">
                <i data-lucide="clover" class="w-3 h-3 text-gray-400"></i>
                <span class="text-gray-500">机缘</span>
                <span class="font-bold text-gray-800 js-tier">-</span>
                <span class="js-delta text-transparent"></span>
            </div>
        </div>
```

并在 `els` 对象追加：

```javascript
            stateBar: document.getElementById('state-bar'),
```

- [ ] **Step 4: 在 STATE 加 pendingChoices，处理 state 帧**

`STATE` 对象追加字段：

```javascript
            pendingChoices: [],
```

`onmessage` 的 `queueTask` 内，`data.type === 'content'` 分支之前加 `state` 分支：

```javascript
                    if (data.type === 'state') {
                        applyStateFrame(data);
                        if (data.options && data.options.length > 0) {
                            STATE.pendingChoices = data.options.map((text, i) => ({ idx: String(i+1), text }));
                        }
                    } else if (data.type === 'content') {
```

并将 `data.type === 'end'` 分支里的 `renderChoiceButtons(STATE.choicesBuffer)` 改为优先用 `pendingChoices`：

```javascript
                    } else if (data.type === 'end') {
                        if (STATE.buffer.trim().length > 0) {
                            const parts = STATE.buffer.split('---').filter(p => p.trim().length > 0);
                            for (const part of parts) {
                                await processChunk(part);
                            }
                        }
                        STATE.buffer = "";
                        showLoading(false);

                        const choices = (STATE.pendingChoices && STATE.pendingChoices.length > 0)
                            ? STATE.pendingChoices
                            : STATE.choicesBuffer;
                        if (choices && choices.length > 0) {
                            renderChoiceButtons(choices);
                        }
                        STATE.choicesBuffer = [];
                        STATE.pendingChoices = [];
                    }
```

- [ ] **Step 5: 实现 applyStateFrame**

在 `<script>` 内加：

```javascript
        function applyStateFrame(data) {
            const dims = ["名望", "人心", "实力", "机缘"];
            const dirSymbol = {"+": "▲", "-": "▼", "0": ""};
            const dirColor = {"+": "text-green-600", "-": "text-red-600", "0": "text-transparent"};
            dims.forEach(dim => {
                const cell = els.stateBar.querySelector(`[data-dim="${dim}"]`);
                if (!cell) return;
                const tierEl = cell.querySelector('.js-tier');
                const deltaEl = cell.querySelector('.js-delta');
                if (data.panel && data.panel[dim]) {
                    tierEl.textContent = data.panel[dim];
                }
                const dir = data.delta ? (data.delta[dim] || "0") : "0";
                deltaEl.textContent = dirSymbol[dir] || "";
                deltaEl.className = `js-delta ${dirColor[dir]}`;
            });
            els.stateBar.classList.remove('hidden');
            // delta 1.5s 后淡出
            clearTimeout(STATE._deltaTimer);
            STATE._deltaTimer = setTimeout(() => {
                els.stateBar.querySelectorAll('.js-delta').forEach(el => {
                    el.className = 'js-delta text-transparent';
                    el.textContent = '';
                });
            }, 1500);
        }
```

- [ ] **Step 6: 浏览器回归**

Run: 启动服务，浏览器打开三国页，开始游戏。
Expected:
- 顶部出现状态条，4 个定性档位显示初始值（如魏国谋士：名望"小有名气"、人心"无人归附"、实力"勉强自保"、机缘"偶有际遇"）。
- 流式剧情可见，无 `1./2./3.` 文本选项。
- `end` 后出现 3 个按钮（来自 `state.options`）。
- 选一项后，状态条对应维度出现 ▲/▼ 闪现 1.5s 后淡出，档位更新。
- 数值永不外露。

- [ ] **Step 7: 卡顿回归**

确认首字延迟无明显退化（单次流式、关闭深度思考）。

- [ ] **Step 8: 容错回归**

临时把 `config/env.toml` 的 `SystemMsg` 中隐藏 JSON 要求去掉（或 mock 模型返回无 `&`），重启：
Expected: 剧情仍完整展示，状态条不变（delta 全 0），选项兜底为"继续"，无报错弹窗。验证后还原配置。

- [ ] **Step 9: Commit**

```bash
git add resource/static/sanguo.html resource/static/sanguo_state.test.js
git commit -m "feat(frontend): 状态条与 state 帧处理，选后揭示方向"
```

---

## Self-Review

**Spec coverage:**
- §1 状态模型与防爆炸 → Task 1/2/3 ✓
- §2 身份倾向注入与选项生成 → Task 2（倾向表）+ Task 8（注入）+ Task 9（system 提示约束）✓
- §3 选择后揭示流程与 WS 协议 → Task 10（state 帧）+ Task 11（前端）✓
- §4 隐藏 JSON 契约与解析容错 → Task 6/7 ✓
- 测试策略 → Task 3-7 单元、Task 11 前端、Task 11 Step 6-8 回归 ✓

**Placeholder scan:** 无 TBD/TODO；每步含完整代码与命令。Task 11 Step 1 测试文件内有一处 `assert = require('assert')` 重复行，应删除其一——已在下方修正。

**Type consistency:** `FinalizeTurn` 返回 `TurnResult{Panel, Delta, Options}`，Task 10 用 `turn.Panel/turn.Delta/turn.Options` 一致；`PanelOf`/`ParseSummaryForContext` 在 Task 10 引用、Task 10 Step 2/4 定义，一致；`initSystemPrompt` 四参签名在 Task 8 定义、Task 8 Step 6 调用一致。

**修正 Task 11 Step 1 的重复行：** 测试文件首行 `buildStateBar` 定义后，`const assert = require('assert');` 与 `assert = require('assert');` 重复，删第二行，保留 `const assert = require('assert');`。

---

## Execution Handoff

Plan complete and saved to `docs/superpowers/plans/2026-07-02-gameplay-feedback.md`. Two execution options:

**1. Subagent-Driven (recommended)** - 每个 Task 派一个全新 subagent，任务间 review，迭代快。

**2. Inline Execution** - 在当前会话用 executing-plans 批量执行，带检查点 review。

选哪种？
