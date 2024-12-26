package bot

import (
	"sync"

	"gotg-bot/db"
)

var (
	userStates = make(map[int64]*db.FSMVoices)
	mu         sync.Mutex
)

// SetFSMState updates the FSM state for a specific user.
func SetFSMState(userID int64, state *db.FSMVoices) {
	mu.Lock()
	defer mu.Unlock()
	userStates[userID] = state
	return
}

// GetFSMState retrieves the FSM state for a specific user. If no state exists, it initializes a new one.
func GetFSMState(userID int64) *db.FSMVoices {
	
	mu.Lock()
	defer mu.Unlock()
	if state, exists := userStates[userID]; exists {
		return state
	}
	state := &db.FSMVoices{}
	userStates[userID] = state
	return state
}

// SetUserState updates the current state of a user.
func SetUserState(userID int64, state string) {
	mu.Lock()
	defer mu.Unlock()
	if _, exists := userStates[userID]; !exists {
		userStates[userID] = &db.FSMVoices{}
	}
	userStates[userID].State = state
}

// GetUserState retrieves the current state of a user.
func GetUserState(userID int64) string {
	mu.Lock()
	defer mu.Unlock()
	if state, exists := userStates[userID]; exists {
		return state.State
	}
	return ""
}
