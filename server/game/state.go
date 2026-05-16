package game

type GameState struct {
	ColorToMove      Color
	CurrentBudget    int
	LeftoverBudget   [ColorLength]int
	EnPassant        Coordinates
	TurnStarted      bool
	MovedThisTurn    bool
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
		MovedThisTurn:    false,
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
