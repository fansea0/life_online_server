package game

import (
	"encoding/json"
	"strings"

	"github.com/sirupsen/logrus"
)

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

// 回合类型阈值常量（便于调参与测试）
const (
	crisisLow     = 1 // 维度 <=此值视为濒危，触发 crisis
	crisisHigh    = 6 // 维度 >=此值视为高维"树大招风"
	boonThreshold = 6 // 维度跨到此值视为刚跨阈值，触发 boon
	timeskipGap   = 3 // 连续此轮数无 timeskip 则候选 timeskip
	crisisGap     = 2 // 连续此轮数无 crisis 则可触发 crisis（需高维）
	boonGap       = 2 // 连续此轮数无 boon 则可触发 boon（需无 crisis 候选）
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

// decideKindCandidate 按状态+历史算候选回合类型。优先级：crisis > timeskip > boon > normal。
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
	if !lastIsCrisis && len(recentKinds) >= timeskipGap && lastKindGapSatisfied(recentKinds, timeskipGap, "timeskip") {
		return "timeskip"
	}
	// 规则3：boon（需历史足够，避免开局即 boon）
	if len(recentKinds) >= boonGap && lastKindGapSatisfied(recentKinds, boonGap, "boon") {
		return "boon"
	}
	return "normal"
}

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

var stateHints = map[string]string{
	"名望": "你如今颇有声名，言行皆为各方关注",
	"人心": "你已深得人心，众人愿为你效力",
	"实力": "你如今手握兵权，行事更受各方瞩目",
	"机缘": "你近来机缘不断，似有天意相助",
}

// stateHintFromState 高维度（>=5）态势提示，多维以；连接，全低返回空串
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
		// 完全无隐藏尾，兜底"继续"避免玩家卡住
		return HiddenPayload{Options: []string{"继续"}}
	}
	var r rawHiddenPayload
	if err := json.Unmarshal([]byte(raw), &r); err != nil {
		logrus.WithError(err).Warn("parseHiddenPayload: unmarshal failed, fallback")
		// JSON 整体解析失败，兜底"继续"
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

// normalizeOptions 保留 AI 实际给出的选项，截断到 3 个，过滤空串。
// 不再用"继续"补足——部分成功就按实际数量返回，避免无故冒出"继续"。
func normalizeOptions(opts []string) []string {
	result := make([]string, 0, 3)
	for _, o := range opts {
		if len(result) >= 3 {
			break
		}
		if s := strings.TrimSpace(o); s != "" {
			result = append(result, s)
		}
	}
	return result
}

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

// ParseSummaryForContext 从隐藏尾解析 summary 文本，供上下文回灌；解析失败返回空串
func ParseSummaryForContext(hiddenRaw string) string {
	return parseHiddenPayload(hiddenRaw).Summary
}

// PanelOf 取会话当前面板，无状态返回 nil
func PanelOf(sessionID string) map[string]string {
	state := GetState(sessionID)
	if state == nil {
		return nil
	}
	return panelFromState(state)
}
