package game

func GetPawnMoves(b *Board, from Coordinates, color Color, enPassant Coordinates) []Coordinates {
	var moves []Coordinates

	direction := -1 // białe idą w górę (malejący row)
	if color == Black {
		direction = 1
	}

	// ruch o jedno pole do przodu
	oneStep := Coordinates{Row: from.Row + direction, Col: from.Col}
	if oneStep.IsValid() && b.IsEmpty(oneStep) {
		moves = append(moves, oneStep)

		// ruch o dwa pola z pozycji startowej
		twoStep := Coordinates{Row: from.Row + 2*direction, Col: from.Col}
		piece := b.Get(from)
		if piece != nil && piece.Piece().FirstMove && b.IsEmpty(twoStep) {
			moves = append(moves, twoStep)
		}
	}

	// bicia skosem
	for _, colOffset := range [2]int{-1, 1} {
		target := Coordinates{Row: from.Row + direction, Col: from.Col + colOffset}
		if !target.IsValid() {
			continue
		}
		if b.HasEnemy(target, color) {
			moves = append(moves, target)
		}
		// en passant - enPassant to pole docelowe (za pionkiem przeciwnika)
		if target.Equals(enPassant) {
			capturedSq := Coordinates{Row: target.Row - direction, Col: target.Col}
			if b.HasEnemy(capturedSq, color) {
				moves = append(moves, target)
			}
		}
	}

	return FilterIllegalMoves(b, from, moves, color)
}
