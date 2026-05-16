package game

func FilterIllegalMoves(b *Board, from Coordinates, candidates []Coordinates, color Color) []Coordinates {
	var legal []Coordinates
	for _, to := range candidates {
		clone := b.Clone()
		piece := clone.Get(from)
		clone.Remove(from)
		clone.Set(to, piece)
		if !KingInCheck(clone, color) {
			legal = append(legal, to)
		}
	}
	return legal
}

func KingInCheck(b *Board, color Color) bool {
	kingPos := b.KingPosition(color)
	if !kingPos.IsValid() {
		return false
	}
	return IsSquareAttacked(b, kingPos, color)
}

func IsSquareAttacked(b *Board, c Coordinates, color Color) bool {
	return IsAttackedByRookOrQueen(b, c, color) ||
		IsAttackedByBishopOrQueen(b, c, color) ||
		IsAttackedByKnight(b, c, color) ||
		IsAttackedByPawn(b, c, color) ||
		IsAttackedByKing(b, c, color)
}

func IsAttackedByBishopOrQueen(b *Board, c Coordinates, color Color) bool {
	directions := [4][2]int{{1, 1}, {1, -1}, {-1, 1}, {-1, -1}}
	for _, d := range directions {
		for i := 1; i < BoardSize; i++ {
			target := Coordinates{Row: c.Row + d[0]*i, Col: c.Col + d[1]*i}
			if !target.IsValid() {
				break
			}
			piece := b.Get(target)
			if piece == nil {
				continue
			}
			if piece.Piece().Color != color && (piece.Type() == BishopType || piece.Type() == QueenType) {
				return true
			}
			break
		}
	}
	return false
}

func IsAttackedByRookOrQueen(b *Board, c Coordinates, color Color) bool {
	directions := [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}}
	for _, d := range directions {
		for i := 1; i < BoardSize; i++ {
			target := Coordinates{Row: c.Row + d[0]*i, Col: c.Col + d[1]*i}
			if !target.IsValid() {
				break
			}
			piece := b.Get(target)
			if piece == nil {
				continue
			}
			if piece.Piece().Color != color && (piece.Type() == RookType || piece.Type() == QueenType) {
				return true
			}
			break
		}
	}
	return false
}

func IsAttackedByKnight(b *Board, c Coordinates, color Color) bool {
	offsets := [8][2]int{{2, 1}, {2, -1}, {-2, 1}, {-2, -1}, {1, 2}, {1, -2}, {-1, 2}, {-1, -2}}
	for _, o := range offsets {
		target := Coordinates{Row: c.Row + o[0], Col: c.Col + o[1]}
		if !target.IsValid() {
			continue
		}
		piece := b.Get(target)
		if piece != nil && piece.Piece().Color != color && piece.Type() == KnightType {
			return true
		}
	}
	return false
}

func IsAttackedByKing(b *Board, c Coordinates, color Color) bool {
	offsets := [8][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}, {1, 1}, {1, -1}, {-1, 1}, {-1, -1}}
	for _, o := range offsets {
		target := Coordinates{Row: c.Row + o[0], Col: c.Col + o[1]}
		if !target.IsValid() {
			continue
		}
		piece := b.Get(target)
		if piece != nil && piece.Piece().Color != color && piece.Type() == KingType {
			return true
		}
	}
	return false
}

func IsAttackedByPawn(b *Board, c Coordinates, color Color) bool {
	// pionek atakuje skosem do przodu - szukamy pionków przeciwnika atakujących to pole
	direction := 1 // pionek białych bije w górę (rosnący row)
	if color == White {
		direction = -1 // szukamy czarnych pionków które biją w dół
	}
	offsets := [2]int{-1, 1}
	for _, colOffset := range offsets {
		target := Coordinates{Row: c.Row + direction, Col: c.Col + colOffset}
		if !target.IsValid() {
			continue
		}
		piece := b.Get(target)
		if piece != nil && piece.Piece().Color != color && piece.Type() == PawnType {
			return true
		}
	}
	return false
}
