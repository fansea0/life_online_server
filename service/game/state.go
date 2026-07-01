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
