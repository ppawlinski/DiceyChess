package game

func GetCastlingMoves(b *Board, from Coordinates, color Color) []Coordinates {
	var moves []Coordinates

	king := b.Get(from)
	if king == nil || !king.Piece().FirstMove {
		return moves
	}

	// król nie może być szachowany przed roszadą
	if KingInCheck(b, color) {
		return moves
	}

	// roszada krótka (kingside)
	rookCol := BoardSize - 1
	if canCastle(b, from, rookCol, color) {
		moves = append(moves, Coordinates{Row: from.Row, Col: from.Col + 2})
	}

	// roszada długa (queenside)
	rookCol = 0
	if canCastle(b, from, rookCol, color) {
		moves = append(moves, Coordinates{Row: from.Row, Col: from.Col - 2})
	}

	return moves
}

func canCastle(b *Board, kingFrom Coordinates, rookCol int, color Color) bool {
	rook := b.Get(Coordinates{Row: kingFrom.Row, Col: rookCol})
	if rook == nil || rook.Type() != RookType || !rook.Piece().FirstMove {
		return false
	}

	// kierunek od króla do wieży
	direction := 1
	if rookCol < kingFrom.Col {
		direction = -1
	}

	// sprawdź czy pola między królem a wieżą są puste
	// i czy król nie przechodzi przez pole atakowane
	for col := kingFrom.Col + direction; col != rookCol; col += direction {
		target := Coordinates{Row: kingFrom.Row, Col: col}
		if !b.IsEmpty(target) {
			return false
		}
		// król przechodzi przez te pola (tylko dwa pierwsze)
		if col == kingFrom.Col+direction || col == kingFrom.Col+2*direction {
			if IsSquareAttacked(b, target, color) {
				return false
			}
		}
	}

	return true
}
