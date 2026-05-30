package game

type Game struct {
	Board   *Board
	State   *GameState
	Dice    *Dice
	turn    *Turn
	WhiteID int64
	BlackID int64
}

func NewGame(seed int64) *Game {
	board := NewBoard()
	board.Reset()
	state := NewGameState()
	dice := NewDice(seed)
	turn := NewTurn(state, board, dice)

	return &Game{
		Board: board,
		State: state,
		Dice:  dice,
		turn:  turn,
	}
}

type MoveRequest struct {
	From Coordinates
	To   Coordinates
}

type PromoteRequest struct {
	At        Coordinates
	PromoteTo PieceType
}

func (g *Game) StartTurn() {
	g.turn.Start()
}

func (g *Game) GetLegalMoves(from Coordinates) ([]Coordinates, error) {
	piece := g.Board.Get(from)
	if piece == nil {
		return nil, ErrNoPieceAtSource
	}
	if piece.Piece().Color != g.State.ColorToMove {
		return nil, ErrNotYourTurn
	}
	isInCheck := KingInCheck(g.Board, g.State.ColorToMove)
	if !g.State.HasBudgetFor(piece.Type(), isInCheck) {
		return nil, ErrInsufficientBudget
	}
	if g.State.CapturedThisTurn[from] {
		return nil, ErrAlreadyCaptured
	}
	return piece.GetPossibleMoves(g.Board, from, g.State.EnPassant), nil
}

func (g *Game) MakeMove(req MoveRequest) error {
	if g.State.IsOver {
		return ErrGameOver
	}

	piece := g.Board.Get(req.From)
	if piece == nil {
		return ErrNoPieceAtSource
	}
	if piece.Piece().Color != g.State.ColorToMove {
		return ErrNotYourTurn
	}

	if g.State.CapturedThisTurn[req.From] {
		return ErrAlreadyCaptured
	}

	isInCheck := KingInCheck(g.Board, g.State.ColorToMove)

	// pierwszy ruch gdy król musi wyjść z szacha
	if isInCheck {
		legalMoves := piece.GetPossibleMoves(g.Board, req.From, g.State.EnPassant)
		found := false
		for _, m := range legalMoves {
			if m.Equals(req.To) {
				found = true
				break
			}
		}
		if !found {
			return ErrMustEscapeCheck
		}
	}

	// sprawdź legalność ruchu
	legalMoves := piece.GetPossibleMoves(g.Board, req.From, g.State.EnPassant)
	found := false
	for _, m := range legalMoves {
		if m.Equals(req.To) {
			found = true
			break
		}
	}
	if !found {
		return ErrIllegalMove
	}

	if !g.State.HasBudgetFor(piece.Type(), isInCheck) {
		return ErrInsufficientBudget
	}

	// zablokuj ruch który cofałby planszę do stanu sprzed tury, gdy gracz mógłby jeszcze zagrać
	if g.State.TurnStartBoardHash != "" && g.simulateMoveHash(req) == g.State.TurnStartBoardHash {
		projectedBudget := g.State.Budgets[g.State.ColorToMove] - EffectiveCost(piece.Type(), isInCheck)
		if g.turn.HasAnyAffordableMoveForColor(projectedBudget, piece.Piece().Color) {
			return ErrWouldRevertBoard
		}
	}

	// wykonaj ruch
	captured := g.Board.Get(req.To)
	if captured != nil {
		g.State.CapturedThisTurn[req.To] = true
	}

	// en passant - zbij pionka
	if piece.Type() == PawnType && req.To.Equals(g.State.EnPassant) {
		direction := 1
		if piece.Piece().Color == White {
			direction = -1
		}
		g.Board.Remove(Coordinates{Row: req.To.Row - direction, Col: req.To.Col})
		g.State.CapturedThisTurn[req.To] = true
	}

	// roszada - przesuń wieżę
	if piece.Type() == KingType && abs(req.To.Col-req.From.Col) == 2 {
		rookFromCol := BoardSize - 1
		rookToCol := req.From.Col + 1
		if req.To.Col < req.From.Col {
			rookFromCol = 0
			rookToCol = req.From.Col - 1
		}
		rook := g.Board.Get(Coordinates{Row: req.From.Row, Col: rookFromCol})
		g.Board.Remove(Coordinates{Row: req.From.Row, Col: rookFromCol})
		g.Board.Set(Coordinates{Row: req.From.Row, Col: rookToCol}, rook)
	}

	// ustaw en passant
	g.State.EnPassant = InvalidCoordinates
	if piece.Type() == PawnType && abs(req.To.Row-req.From.Row) == 2 {
		direction := 1
		if piece.Piece().Color == White {
			direction = -1
		}
		g.State.EnPassant = Coordinates{Row: req.From.Row + direction, Col: req.From.Col}
	}

	piece.Move(req.To)
	g.Board.Remove(req.From)
	g.Board.Set(req.To, piece)

	g.State.SpendBudget(piece.Type(), isInCheck)
	if g.State.Budgets[g.State.ColorToMove] == 0 {
		g.EndTurn()
	}

	return nil
}

func (g *Game) simulateMoveHash(req MoveRequest) string {
	clone := g.Board.Clone()
	piece := clone.Get(req.From)
	if piece == nil {
		return ""
	}
	if piece.Type() == PawnType && req.To.Equals(g.State.EnPassant) {
		direction := 1
		if piece.Piece().Color == White {
			direction = -1
		}
		clone.Remove(Coordinates{Row: req.To.Row - direction, Col: req.To.Col})
	}
	if piece.Type() == KingType && abs(req.To.Col-req.From.Col) == 2 {
		rookFromCol := BoardSize - 1
		rookToCol := req.From.Col + 1
		if req.To.Col < req.From.Col {
			rookFromCol = 0
			rookToCol = req.From.Col - 1
		}
		rook := clone.Get(Coordinates{Row: req.From.Row, Col: rookFromCol})
		clone.Remove(Coordinates{Row: req.From.Row, Col: rookFromCol})
		clone.Set(Coordinates{Row: req.From.Row, Col: rookToCol}, rook)
	}
	clone.Remove(req.From)
	clone.Set(req.To, piece)
	return clone.Hash()
}

func (g *Game) CanAffordToMove() bool {
	return g.turn.CanAffordToMove()
}

func (g *Game) EndTurn() error {
	g.turn.End()
	// sprawdź mat/pat dla następnego gracza
	opponent := g.State.ColorToMove // po End() to już następny gracz
	if !g.turn.hasAnyLegalMoveForColor(opponent) {
		g.State.IsOver = true
		if KingInCheck(g.Board, opponent) {
			winner := White
			if opponent == White {
				winner = Black
			}
			g.State.Winner = &winner
		}
		// pat - Winner zostaje nil
	}
	return nil
}

func (g *Game) Promote(req PromoteRequest) error {
	if g.State.IsOver {
		return ErrGameOver
	}

	piece := g.Board.Get(req.At)
	if piece == nil || piece.Type() != PawnType {
		return ErrNotAPromotablePawn
	}
	if piece.Piece().Color != g.State.ColorToMove {
		return ErrNotYourTurn
	}
	if !IsPromotionSquare(req.At, piece.Piece().Color) {
		return ErrNotAPromotablePawn
	}
	if !CanPromoteTo(req.PromoteTo) {
		return ErrInvalidPromotion
	}
	if !g.State.HasBudgetFor(req.PromoteTo, false) {
		return ErrInsufficientBudget
	}

	g.State.SpendBudget(req.PromoteTo, false)
	HandlePromotion(g.Board, req.At, piece.Piece().Color, req.PromoteTo)

	if g.State.Budgets[g.State.ColorToMove] == 0 {
		g.EndTurn()
	}

	return nil
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
