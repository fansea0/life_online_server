# 回合类型轮转设计

## 背景与问题

前一版反馈优化（见 `2026-07-02-gameplay-feedback-design.md`）落地后，玩法"不好玩"的核心卡点是**节奏单调**：每回合都是"长叙事 + 3 选项"的复刻，结构同质，缺乏起伏与换气。

本设计在已有状态面板/隐藏 JSON/选后揭示之上，引入**回合类型差异**，让相邻回合"长得不一样"。

## 目标

- 打破"每回合同结构"的单调感，让节奏有起伏（紧张/高潮/换气）。
- 类型判定服务端兜底 + AI 演绎，避免纯靠 AI 调节节奏而漂移。
- 复用现有 `&` 隐藏通道、`state` 帧、`GameState`，最小改动。
- 任何类型字段缺失/非法都不阻断剧情，降级为常态。

## 非目标

- 不改既有 4 维状态模型与防爆炸机制。
- 不改选后揭示原则（不露数值）。
- 不引入存档、多周目、成就。
- 不做自由输入行动。

## 方案

### §1 4 种回合类型与判定

**TurnKind 单选，每轮一个：**

| 类型 | 触发条件 | 选项数 | 叙事长度 | 情绪 |
|---|---|---|---|---|
| `normal` 常态抉择 | 默认，无特殊触发 | 3 | 中（现有） | 平稳推进 |
| `crisis` 突发危机 | 某维度 ≤1（濒临崩溃），或连续 ≥2 轮无 crisis 且状态有 ≥6 高维（树大招风） | 2，各有代价 | 短急 | 紧张 |
| `boon` 机缘偶得 | 某维度刚跨过阈值（5→6 等），或连续 ≥2 轮无 boon 且无危机候选 | 3，无负面 delta | 短喜 | 高潮 |
| `timeskip` 时过境迁 | 连续 ≥3 轮无 timeskip，且最近一轮非 crisis | 2（跳过/介入） | 极简概括+时间标注 | 换气 |

**判定优先级（服务端决定，不全交给 AI）：**
1. 服务端按 `GameState` + `RecentKinds` 算出候选类型。
2. 候选写进 system 上下文，AI 必须按候选类型生成。
3. AI 实际输出 `kind` 若与服务端候选不符，以 AI 输出为准（AI 据剧情合理性微调），记 warn。
4. 任何缺失/非法 → 降级 `normal`。

**为什么服务端参与：** 纯靠 AI 自我调节节奏会漂移（要么一直 crisis、要么一直 normal）。服务端用"连续 N 轮无 X 则补 X"的简单计数器兜住节拍，AI 负责把节拍演绎成具体剧情。

**历史记录：** `GameState` 加 `RecentKinds []string`（最近 5 轮类型），服务端据此算候选。纯函数、可测。

### §2 隐藏 JSON 扩展与服务端判定

**隐藏 JSON 增加两字段：**

```json
{
  "kind": "normal",
  "summary": "...",
  "state_delta": {"名望":0,"人心":0,"实力":0,"机缘":0},
  "options": ["...","...","..."]
}
```

- `kind`：AI 实际采用的回合类型，与候选可能不同。
- 其余不变。

**服务端判定函数（纯函数）：**

`func decideKindCandidate(s *GameState, recentKinds []string) string`

规则（按优先级，命中即返回）：
1. **timeskip** — recentKinds 连续 ≥3 轮无 timeskip，且最近一轮非 crisis → timeskip。
2. **crisis** — 任一维度 ≤1，或（连续 ≥2 轮无 crisis 且存在维度 ≥6）→ crisis。
3. **boon** — 上一轮有维度刚好跨阈值（5→6 或更高），或（连续 ≥2 轮无 boon 且无 crisis 候选）→ boon。
4. **normal** — 兜底。

阈值常量集中定义（便于调参与测试）：
- `crisisLow = 1`、`crisisHigh = 6`、`boonThreshold = 6`
- `timeskipGap = 3`、`crisisGap = 2`、`boonGap = 2`

**候选与实际协调：**
- 服务端把 candidate 写进 system 上下文："本轮建议回合类型：{candidate}（{规则说明}）。你必须按此类型生成剧情与选项，并在 kind 字段回填实际类型。"
- `parseHiddenPayload` 解析 `kind`，校验是否在 4 个合法值内，非法/缺失 → normal，记 warn。
- `FinalizeTurn` 把 AI 实际 `kind` 追加进 `state.RecentKinds`（保留最近 5 个），持久化。

**判定时机：** `HandleChoiceStream` 注入 system 前算 candidate；开局首轮（`StartGameStream`）强制 normal，RecentKinds 为空。

### §3 各类型 prompt 约束与前端呈现

**system 提示按候选类型追加规则段（服务端拼接）：**

```
[normal] 叙事100-150字，给出3个风格分化的选项，可含正负后果。
[crisis]  叙事≤60字，急促；只给2个选项，两者都有代价（必有负面delta）；无第三选项。
[boon]    叙事≤80字，喜悦基调；3个选项均无负面delta，让玩家选偏向哪个维度的收益。
[timeskip] 叙事≤40字概括数月/数年经过 + 一句"时过境迁"；2个选项（"就此翻过"/"期间有所动作"）；本类型state_delta可更大（±2常态）。
```

选项数约束写死在规则里，AI 不该越权。服务端不二次裁剪选项数（信任 prompt + 记 warn），避免砍掉 AI 给的有效选项。

**前端按 kind 差异化呈现（state 帧带 kind）：**

| kind | 视觉 | 选项区 |
|---|---|---|
| normal | 现状 | 3 按钮 |
| crisis | 顶部红色细条 + 叙事卡红框 + "危急"标签 | 2 按钮，红边 |
| boon | 叙事卡金色边 + "机缘"标签 | 3 按钮，金边 |
| timeskip | 横向时间轴分隔线 + "数月/数年后"居中标注 | 2 按钮，灰边 |

**state 帧扩展：**
```json
{"type":"state","kind":"crisis","panel":{...},"delta":{...},"options":["...","..."]}
```
前端 `applyStateFrame` 读 kind，给叙事容器加临时 class（`turn-crisis`/`turn-boon`/`turn-timeskip`/`turn-normal`），下回合开始时移除。

**叙事卡识别：** 现有 `renderChunk` 按 `#`/`>`/对话 分流。新增：`kind=timeskip` 时第一段 `#` 标题渲染为时间轴分隔（居中+横线），而非普通标题。其余 kind 不改 `renderChunk`，只靠容器边框色区分，改动最小。

**回退：** kind 缺失/非法 → 前端按 normal 呈现。

### §4 容错与边界

1. AI 未输出 `kind` → 降级 normal，记 warn。
2. AI 输出非法值 → 降级 normal，记 warn。
3. AI 输出 `kind` 与候选不符 → 以 AI 为准（据剧情微调合理），记 warn。不强制纠正。
4. AI 在 crisis/timeskip 下给 3 选项 → 不裁剪，记 warn。前端照常渲染。下次调 prompt。
5. timeskip 的 state_delta 超 ±2 → 仍忽略越界值（防爆炸不放松），记 warn。

**RecentKinds 维护：**
- `FinalizeTurn` 追加 AI 实际 kind，超 5 个截断（保留最近 5）。
- `decideKindCandidate` 读 state.RecentKinds，不足 5 用现有长度判定（不补默认）。
- 开局 RecentKinds 为空 → 候选强制 normal。

**前端容错：** state 帧无 kind → normal；kind 不认识 → normal；新 state 帧先清除上轮容器 class 再加新。

**剧情断裂：** timeskip 规则已用"最近一轮 crisis 则不 timeskip"兜住；AI 仍觉不合理可改 kind 回 normal/crisis，服务端不拦。

## 数据流

```text
玩家选择
  |
  +-- 后端读 GameState + RecentKinds
        |
        +-- decideKindCandidate -> candidate
              |
              +-- 注入 system：候选类型 + 该类型规则段
                    |
                    +-- 单次流式（关闭深度思考、有限超时）
                          |
                          +-- & 之前 -> content 帧（剧情，按 kind 差异化呈现）
                          |
                          +-- & 之后 -> 累积 JSON（含 kind）
                                |
                                +-- 流结束: parse(kind 校验) -> applyDelta -> 持久化 RecentKinds
                                      |
                                      +-- state 帧（kind + panel + delta + options）
                                      |
                                      +-- end 帧
```

## 组件改动

- `service/game/types.go`：`GameState` 加 `RecentKinds []string`。
- `service/game/state.go`：加阈值常量、`decideKindCandidate`、`kindPromptRule(kind)`、`TurnKind` 校验；`parseHiddenPayload` 解析 `kind`；`FinalizeTurn` 维护 `RecentKinds`；`TurnResult` 加 `Kind`。
- `service/game/game.go`：`HandleChoiceStream` 算 candidate 并拼进 system；`StartGameStream` 首轮 normal。
- `route/game/game.go`：`state` 帧带 `kind`。
- `config/env.toml`：`SystemMsg` 增加 kind 候选占位 `{{.turnKindRule}}` 与说明。
- `resource/static/sanguo.html`：`applyStateFrame` 读 kind 加容器 class；timeskip 标题时间轴渲染；上轮 class 清除。
- 测试：`state_test.go`（decideKindCandidate、kind 解析、RecentKinds 维护）；`sanguo_state.test.js`（kind→class 映射）。

## 测试

**单元测试（Go）：**
1. `decideKindCandidate`：4 规则各自触发、优先级冲突（timeskip vs crisis → crisis）、recentKinds 不足 5（空/1/3）。
2. `parseHiddenPayload`：合法 4 值 / 非法降级 normal / 缺失降级 normal。
3. `FinalizeTurn`：kind 入 RecentKinds、超 5 截断、首轮空→normal。
4. 阈值常量值校验（防误改）。

**前端测试：**
- `applyStateFrameKind`：kind→class 映射、缺失/非法→normal、上轮 class 清除。

**回归（人工，浏览器）：**
- 连续 6+ 轮观察类型轮转（normal×3→timeskip；危机冒 crisis；跨阈值冒 boon）。
- crisis 红框+2选项；boon 金框；timeskip 时间轴+2选项。
- kind 缺失降级 normal 不报错。
- 数值不外露、防爆炸仍生效。

**不测：** AI 剧情质量、类型判定手感（需人工体验调参）。