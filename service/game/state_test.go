package game

import (
	"strings"
	"testing"
)

func TestLookupIdentity_Known(t *testing.T) {
	init := LookupIdentity("魏国的一名谋士")
	if init.IdentityAffinity == "" {
		t.Error("IdentityAffinity should not be empty for known identity")
	}
	if init.Attributes["名望"] != 1 {
		t.Errorf("名望 = %d, want 1", init.Attributes["名望"])
	}
}

func TestLookupIdentity_Unknown(t *testing.T) {
	init := LookupIdentity("不存在的身份")
	if init.IdentityAffinity != defaultIdentityInit.IdentityAffinity {
		t.Error("Should return default identity for unknown identity")
	}
}

func TestCreateNewGame(t *testing.T) {
	state := CreateNewGame("测试玩家", "蜀国的一名年轻将领")
	if state.Name != "测试玩家" {
		t.Errorf("Name = %q, want %q", state.Name, "测试玩家")
	}
	if state.Identity != "蜀国的一名年轻将领" {
		t.Errorf("Identity = %q, want %q", state.Identity, "蜀国的一名年轻将领")
	}
	if state.IdentityAffinity == "" {
		t.Error("IdentityAffinity should not be empty")
	}
	if state.Attributes["名望"] != 1 {
		t.Errorf("名望 = %d, want 1", state.Attributes["名望"])
	}
}

func TestCreateNewGame_Default(t *testing.T) {
	state := CreateNewGame("你", "")
	if state.Identity != "" {
		t.Errorf("Identity = %q, want empty", state.Identity)
	}
	if state.Attributes["实力"] != 1 {
		t.Errorf("实力 = %d, want 1", state.Attributes["实力"])
	}
}

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
	if panel["名望"] != "名震一方" || panel["人心"] != "众心所向" || panel["实力"] != "可堪一用" || panel["机缘"] != "偶有际遇" {
		t.Fatalf("all dims: got %+v", panel)
	}
}

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

func TestParseHiddenPayloadOptionsPartialAndTruncate(t *testing.T) {
	// 不足 3 个时按实际数量返回，不再补"继续"
	p := parseHiddenPayload(`{"options":["only one"]}`)
	if len(p.Options) != 1 || p.Options[0] != "only one" {
		t.Fatalf("partial: got %+v", p.Options)
	}
	// 超过 3 个截断到 3
	p2 := parseHiddenPayload(`{"options":["a","b","c","d","e"]}`)
	if len(p2.Options) != 3 || p2.Options[2] != "c" {
		t.Fatalf("truncate: got %+v", p2.Options)
	}
	// 空串被过滤
	p3 := parseHiddenPayload(`{"options":["a","","c"]}`)
	if len(p3.Options) != 2 || p3.Options[0] != "a" || p3.Options[1] != "c" {
		t.Fatalf("filter blank: got %+v", p3.Options)
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

func TestDecideKindTimeskipAfterGap(t *testing.T) {
	s := &GameState{Attributes: map[string]int{"名望": 3, "人心": 3, "实力": 3, "机缘": 3}}
	recent := []string{"normal", "normal", "normal"}
	if got := decideKindCandidate(s, recent); got != "timeskip" {
		t.Fatalf("timeskip gap: got %q, want timeskip", got)
	}
}

func TestDecideKindTimeskipBlockedByCrisis(t *testing.T) {
	s := &GameState{Attributes: map[string]int{"名望": 3, "人心": 3, "实力": 3, "机缘": 3}}
	recent := []string{"normal", "normal", "crisis"}
	if got := decideKindCandidate(s, recent); got == "timeskip" {
		t.Fatalf("should not timeskip after crisis, got timeskip")
	}
}

func TestDecideKindCrisisLowDim(t *testing.T) {
	s := &GameState{Attributes: map[string]int{"名望": 1, "人心": 5, "实力": 5, "机缘": 5}}
	if got := decideKindCandidate(s, []string{"normal"}); got != "crisis" {
		t.Fatalf("crisis low: got %q, want crisis", got)
	}
}

func TestDecideKindCrisisHighDimAfterGap(t *testing.T) {
	s := &GameState{Attributes: map[string]int{"名望": 6, "人心": 3, "实力": 3, "机缘": 3}}
	if got := decideKindCandidate(s, []string{"normal", "normal"}); got != "crisis" {
		t.Fatalf("crisis high: got %q, want crisis", got)
	}
}

func TestDecideKindPriorityCrisisOverTimeskip(t *testing.T) {
	s := &GameState{Attributes: map[string]int{"名望": 1, "人心": 5, "实力": 5, "机缘": 5}}
	recent := []string{"normal", "normal", "normal"}
	if got := decideKindCandidate(s, recent); got != "crisis" {
		t.Fatalf("priority: got %q, want crisis", got)
	}
}

func TestDecideKindBoonAfterGapNoCrisis(t *testing.T) {
	s := &GameState{Attributes: map[string]int{"名望": 3, "人心": 3, "实力": 3, "机缘": 3}}
	if got := decideKindCandidate(s, []string{"normal", "normal"}); got != "boon" {
		t.Fatalf("boon: got %q, want boon", got)
	}
}

func TestDecideKindNormalEmptyRecent(t *testing.T) {
	s := &GameState{Attributes: map[string]int{"名望": 3, "人心": 3, "实力": 3, "机缘": 3}}
	if got := decideKindCandidate(s, []string{}); got != "normal" {
		t.Fatalf("empty recent: got %q, want normal", got)
	}
}

func TestDecideKindNormalSingleRecent(t *testing.T) {
	s := &GameState{Attributes: map[string]int{"名望": 4, "人心": 4, "实力": 4, "机缘": 4}}
	if got := decideKindCandidate(s, []string{"normal"}); got != "normal" {
		t.Fatalf("single recent: got %q, want normal", got)
	}
}

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
	if kindPromptRule("boss") != kindPromptRule("normal") {
		t.Fatalf("kindPromptRule(boss) should equal normal rule")
	}
}
