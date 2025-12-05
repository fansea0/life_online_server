package game

// GameState 玩家当前状态
type GameState struct {
	SessionID  string         `json:"session_id"`
	Name       string         `json:"name"`
	Age        int            `json:"age"`
	Attributes map[string]int `json:"attributes"` // 例如: {"智力": 10, "体质": 8, "财富": 0}
	Inventory  []string       `json:"inventory"`  // 背包
	Summary    string         `json:"summary"`    // 之前的剧情摘要
	IsGameOver bool           `json:"is_game_over"`
}

// UserAction 用户发送的请求
type UserAction struct {
	SessionID string `json:"session_id"`
	Choice    string `json:"choice"` // 用户选了什么
}

// AIResponse AI返回的结构（JSON部分）
type AIResponse struct {
	Options    []string       `json:"options"`     // 新选项
	AttrChange map[string]int `json:"attr_change"` // 属性变化
	GameOver   bool           `json:"game_over"`   // 是否结束
}

// GameResponse 返回给前端的完整结构
type GameResponse struct {
	Story      string         `json:"story"`
	State      *GameState     `json:"state"`
	Options    []string       `json:"options"`
	AttrChange map[string]int `json:"attr_change"`
}
