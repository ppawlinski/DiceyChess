package game

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

type coordinateKey struct {
	Row int
	Col int
}

type GameState struct {
	ColorToMove      Color
	CurrentBudget    int
	LeftoverBudget   [ColorLength]int
	EnPassant        Coordinates
	TurnStarted      bool
	CapturedThisTurn map[Coordinates]bool
	IsOver           bool
	Winner           *Color // nil jeśli remis
}

func NewGameState() *GameState {
	return &GameState{
		ColorToMove:      White,
		CurrentBudget:    0,
		LeftoverBudget:   [ColorLength]int{0, 0},
		EnPassant:        InvalidCoordinates,
		TurnStarted:      false,
		CapturedThisTurn: make(map[Coordinates]bool),
		IsOver:           false,
		Winner:           nil,
	}
}

func (gs *GameState) HasBudgetFor(pt PieceType, isKingInCheck bool) bool {
	cost := MoveCost(pt)
	if pt == KingType && isKingInCheck {
		cost = CheckedKingCost
	}
	return gs.CurrentBudget >= cost
}

func (gs *GameState) SpendBudget(pt PieceType, isKingInCheck bool) {
	cost := MoveCost(pt)
	if pt == KingType && isKingInCheck {
		cost = CheckedKingCost
	}
	gs.CurrentBudget -= cost
}

func (gs *GameState) MarshalJSON() ([]byte, error) {
	captured := make(map[string]bool)
	for k, v := range gs.CapturedThisTurn {
		key := fmt.Sprintf("%d,%d", k.Row, k.Col)
		captured[key] = v
	}

	type Alias struct {
		ColorToMove      Color
		CurrentBudget    int
		LeftoverBudget   [ColorLength]int
		EnPassant        Coordinates
		TurnStarted      bool
		CapturedThisTurn map[string]bool
		IsOver           bool
		Winner           *Color
	}

	return json.Marshal(Alias{
		ColorToMove:      gs.ColorToMove,
		CurrentBudget:    gs.CurrentBudget,
		LeftoverBudget:   gs.LeftoverBudget,
		EnPassant:        gs.EnPassant,
		TurnStarted:      gs.TurnStarted,
		CapturedThisTurn: captured,
		IsOver:           gs.IsOver,
		Winner:           gs.Winner,
	})
}

func (gs *GameState) UnmarshalJSON(data []byte) error {
	type Alias struct {
		ColorToMove      Color
		CurrentBudget    int
		LeftoverBudget   [ColorLength]int
		EnPassant        Coordinates
		TurnStarted      bool
		CapturedThisTurn map[string]bool
		IsOver           bool
		Winner           *Color
	}

	var a Alias
	if err := json.Unmarshal(data, &a); err != nil {
		return err
	}

	gs.ColorToMove = a.ColorToMove
	gs.CurrentBudget = a.CurrentBudget
	gs.LeftoverBudget = a.LeftoverBudget
	gs.EnPassant = a.EnPassant
	gs.TurnStarted = a.TurnStarted
	gs.IsOver = a.IsOver
	gs.Winner = a.Winner

	gs.CapturedThisTurn = make(map[Coordinates]bool)
	for k, v := range a.CapturedThisTurn {
		parts := strings.Split(k, ",")
		if len(parts) != 2 {
			continue
		}
		row, err1 := strconv.Atoi(parts[0])
		col, err2 := strconv.Atoi(parts[1])
		if err1 != nil || err2 != nil {
			continue
		}
		gs.CapturedThisTurn[Coordinates{Row: row, Col: col}] = v
	}

	return nil
}

func GameStateFromJSON(data string) (*GameState, error) {
	var gs GameState
	if err := json.Unmarshal([]byte(data), &gs); err != nil {
		return nil, err
	}
	return &gs, nil
}
