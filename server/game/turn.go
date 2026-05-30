package game

type Turn struct {
	State *GameState
	Board *Board
	Dice  *Dice
}

func NewTurn(state *GameState, board *Board, dice *Dice) *Turn {
	return &Turn{
		State: state,
		Board: board,
		Dice:  dice,
	}
}

func (t *Turn) Start() {
	roll := t.Dice.Roll()
	t.State.LastRoll = roll
	leftover := t.State.Budgets[t.State.ColorToMove]
	t.State.Budgets[t.State.ColorToMove] = roll + leftover
	t.State.TurnStarted = true
	t.State.CapturedThisTurn = make(map[Coordinates]bool)
	t.State.TurnStartBoardHash = t.Board.Hash()
}

func (t *Turn) CanAffordAnyMove(moves []Coordinates, piece Movable) bool {
	isKingInCheck := t.isKingInCheck()
	return t.State.HasBudgetFor(piece.Type(), isKingInCheck)
}

func (t *Turn) isKingInCheck() bool {
	kingPos := t.Board.KingPosition(t.State.ColorToMove)
	return IsSquareAttacked(t.Board, kingPos, t.State.ColorToMove)
}

func (t *Turn) End() {
	if t.State.Budgets[t.State.ColorToMove] > 0 && t.CanAffordToMove() {
		t.State.Budgets[t.State.ColorToMove] = 0
	}

	// zmiana gracza
	if t.State.ColorToMove == White {
		t.State.ColorToMove = Black
	} else {
		t.State.ColorToMove = White
	}

	/*if !t.hasAnyLegalMove() {
		//!!!3PPA game over
		if t.State.ColorToMove == White {
			t.State.Winner = Black
		}
	}*/

	t.State.TurnStarted = false
}

func (t *Turn) hasAnyLegalMove() bool {
	for row := 0; row < BoardSize; row++ {
		for col := 0; col < BoardSize; col++ {
			c := Coordinates{Row: row, Col: col}
			piece := t.Board.Get(c)
			if piece == nil || piece.Piece().Color != t.State.ColorToMove {
				continue
			}
			moves := piece.GetPossibleMoves(t.Board, c, t.State.EnPassant)
			if len(moves) > 0 {
				return true
			}
		}
	}
	return false
}

func (t *Turn) CanAffordToMove() bool {
	return t.HasAnyAffordableMoveForColor(t.State.Budgets[t.State.ColorToMove], t.State.ColorToMove)
}

func (t *Turn) HasAnyAffordableMoveForColor(budget int, color Color) bool {
	isKingInCheck := KingInCheck(t.Board, color)
	for row := 0; row < BoardSize; row++ {
		for col := 0; col < BoardSize; col++ {
			c := Coordinates{Row: row, Col: col}
			piece := t.Board.Get(c)
			if piece == nil || piece.Piece().Color != color {
				continue
			}
			if budget < EffectiveCost(piece.Type(), isKingInCheck) {
				continue
			}
			if len(piece.GetPossibleMoves(t.Board, c, t.State.EnPassant)) > 0 {
				return true
			}
		}
	}
	return false
}

func (t *Turn) hasAnyLegalMoveForColor(color Color) bool {
	for row := 0; row < BoardSize; row++ {
		for col := 0; col < BoardSize; col++ {
			c := Coordinates{Row: row, Col: col}
			piece := t.Board.Get(c)
			if piece == nil || piece.Piece().Color != color {
				continue
			}
			moves := piece.GetPossibleMoves(t.Board, c, t.State.EnPassant)
			if len(moves) > 0 {
				return true
			}
		}
	}
	return false
}
