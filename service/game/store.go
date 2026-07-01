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

func CreateNewGame() *GameState {
	sessionID := uuid.New().String()
	state := &GameState{
		SessionID: sessionID,
		Name:      "你",
		Age:       0,
		Attributes: map[string]int{
			"名望": 5,
			"人心": 5,
			"实力": 5,
			"机缘": 5,
		},
		Summary: "你出生了，这是一个新的开始。",
	}
	SaveState(state)
	return state
}
