package game

func GetKnightMoves(b *Board, from Coordinates, color Color) []Coordinates {
	var moves []Coordinates

	offsets := [8][2]int{{2, 1}, {2, -1}, {-2, 1}, {-2, -1}, {1, 2}, {1, -2}, {-1, 2}, {-1, -2}}
	for _, o := range offsets {
		target := Coordinates{Row: from.Row + o[0], Col: from.Col + o[1]}
		if !target.IsValid() {
			continue
		}
		if !b.HasFriendly(target, color) {
			moves = append(moves, target)
		}
	}

	return FilterIllegalMoves(b, from, moves, color)
}
