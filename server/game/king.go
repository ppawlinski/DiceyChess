package game

func GetKingMoves(b *Board, from Coordinates, color Color) []Coordinates {
	var candidates []Coordinates

	offsets := [8][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}, {1, 1}, {1, -1}, {-1, 1}, {-1, -1}}
	for _, o := range offsets {
		target := Coordinates{Row: from.Row + o[0], Col: from.Col + o[1]}
		if !target.IsValid() {
			continue
		}
		if b.HasFriendly(target, color) {
			continue
		}
		// król nie może wejść na pole atakowane przez przeciwnika
		if IsSquareAttacked(b, target, color) {
			continue
		}
		candidates = append(candidates, target)
	}

	return FilterIllegalMoves(b, from, candidates, color)
}
