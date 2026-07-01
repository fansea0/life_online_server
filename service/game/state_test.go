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
