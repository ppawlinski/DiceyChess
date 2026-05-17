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
	From      Coordinates
	To        Coordinates
	PromoteTo *PieceType // nil jeśli nie promocja
}

func (g *Game) StartTurn() {
	g.turn.Start()
	if g.turn.SkipIfNecessary() {
		g.turn.Start()
		g.turn.SkipIfNecessary()
	}
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

	isInCheck := KingInCheck(g.Board, g.State.ColorToMove)

	// pierwszy ruch gdy szach musi wyjść z szacha
	if isInCheck && !g.State.MovedThisTurn {
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

	// wykonaj ruch
	captured := g.Board.Get(req.To)
	if captured != nil {
		g.State.CapturedThisTurn[req.From] = true
	}

	// en passant - zbij pionka
	if piece.Type() == PawnType && req.To.Equals(g.State.EnPassant) {
		direction := 1
		if piece.Piece().Color == White {
			direction = -1
		}
		g.Board.Remove(Coordinates{Row: req.To.Row - direction, Col: req.To.Col})
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

	// promocja
	if piece.Type() == PawnType && IsPromotionSquare(req.To, piece.Piece().Color) {
		if req.PromoteTo == nil {
			return ErrInvalidPromotion
		}
		if !CanPromoteTo(*req.PromoteTo) {
			return ErrInvalidPromotion
		}
		promotionCost := MoveCost(*req.PromoteTo)
		if g.State.CurrentBudget < promotionCost {
			return ErrInsufficientBudget
		}
		g.State.SpendBudget(*req.PromoteTo, false)
		HandlePromotion(g.Board, req.To, piece.Piece().Color, *req.PromoteTo)
	}

	g.State.SpendBudget(piece.Type(), isInCheck)
	g.State.MovedThisTurn = true

	// sprawdź mat
	opponent := Black
	if g.State.ColorToMove == Black {
		opponent = White
	}
	if KingInCheck(g.Board, opponent) {
		if !g.turn.hasAnyLegalMove() {
			g.State.IsOver = true
			g.State.Winner = &g.State.ColorToMove
		}
	}

	return nil
}

func (g *Game) EndTurn() error {
	if !g.State.MovedThisTurn {
		return ErrBoardUnchanged
	}
	g.turn.End()
	g.StartTurn()
	return nil
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
