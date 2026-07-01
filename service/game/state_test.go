package game

import "testing"

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
