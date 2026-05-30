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
	Board   string  `json:"board"`
	State   string  `json:"state"`
	Seed    int64   `json:"seed"`
	WhiteID int64   `json:"white_id"`
	BlackID int64   `json:"black_id"`
	History History `json:"history"`
	TurnNum int     `json:"turn_num"`
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
		Board:   string(boardJSON),
		State:   string(stateJSON),
		Seed:    g.Dice.seed,
		WhiteID: g.WhiteID,
		BlackID: g.BlackID,
		History: g.History,
		TurnNum: g.TurnNum,
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

	history := sg.History
	if history == nil {
		history = History{}
	}

	return &Game{
		Board:   board,
		State:   state,
		Dice:    dice,
		turn:    turn,
		WhiteID: sg.WhiteID,
		BlackID: sg.BlackID,
		History: history,
		TurnNum: sg.TurnNum,
	}, nil
}

// SerializeForDB returns the state JSON and human-readable PGN string.
func (g *Game) SerializeForDB() (state string, pgn string, err error) {
	state, err = g.Serialize()
	if err != nil {
		return "", "", err
	}
	pgn = g.History.ToPGN()
	return state, pgn, nil
}

type DBGame struct {
	ID         int64
	FinishedAt *time.Time
	Result     *string
}
