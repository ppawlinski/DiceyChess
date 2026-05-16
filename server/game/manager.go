package game

import (
	"encoding/json"
	"sync"
	"time"
)

type GameManager struct {
	games map[int64]*Game
	mu    sync.RWMutex
}

func NewGameManager() *GameManager {
	return &GameManager{
		games: make(map[int64]*Game),
	}
}

func (gm *GameManager) Add(id int64, g *Game) {
	gm.mu.Lock()
	defer gm.mu.Unlock()
	gm.games[id] = g
}

func (gm *GameManager) Get(id int64) (*Game, bool) {
	gm.mu.RLock()
	defer gm.mu.RUnlock()
	g, ok := gm.games[id]
	return g, ok
}

func (gm *GameManager) Remove(id int64) {
	gm.mu.Lock()
	defer gm.mu.Unlock()
	delete(gm.games, id)
}

type GameSnapshot struct {
	Board *Board
	State *GameState
}

func (gm *GameManager) Snapshot(id int64) (*GameSnapshot, bool) {
	gm.mu.RLock()
	defer gm.mu.RUnlock()
	g, ok := gm.games[id]
	if !ok {
		return nil, false
	}
	return &GameSnapshot{Board: g.Board, State: g.State}, true
}

// SerializedGame to format zapisu do bazy
type SerializedGame struct {
	Board string `json:"board"`
	State string `json:"state"`
	Seed  int64  `json:"seed"`
}

func (g *Game) Serialize() (string, error) {
	boardJSON, err := json.Marshal(g.Board)
	if err != nil {
		return "", err
	}
	stateJSON, err := json.Marshal(g.State)
	if err != nil {
		return "", err
	}
	sg := SerializedGame{
		Board: string(boardJSON),
		State: string(stateJSON),
		Seed:  g.Dice.seed,
	}
	data, err := json.Marshal(sg)
	return string(data), err
}

func DeserializeGame(data string) (*Game, error) {
	var sg SerializedGame
	if err := json.Unmarshal([]byte(data), &sg); err != nil {
		return nil, err
	}

	board := NewBoard()
	if err := json.Unmarshal([]byte(sg.Board), board); err != nil {
		return nil, err
	}

	state, err := GameStateFromJSON(sg.State)
	if err != nil {
		return nil, err
	}

	dice := NewDice(sg.Seed)
	turn := NewTurn(state, board, dice)

	return &Game{
		Board: board,
		State: state,
		Dice:  dice,
		turn:  turn,
	}, nil
}

func (g *Game) SerializeForDB() (string, error) {
	return g.Serialize()
}

type DBGame struct {
	ID         int64
	FinishedAt *time.Time
	Result     *string
}
