package game

import (
	"sync"

	"github.com/google/uuid"
)

var (
	store = make(map[string]*GameState)
	mu    sync.RWMutex
)

func GetState(sessionID string) *GameState {
	mu.RLock()
	defer mu.RUnlock()
	s, ok := store[sessionID]
	if !ok {
		return nil
	}
	return s
}

func SaveState(state *GameState) {
	mu.Lock()
	defer mu.Unlock()
	store[state.SessionID] = state
}

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
