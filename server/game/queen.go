package game

func GetQueenMoves(b *Board, from Coordinates, color Color) []Coordinates {
	var candidates []Coordinates

	directions := [8][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}, {1, 1}, {1, -1}, {-1, 1}, {-1, -1}}
	for _, d := range directions {
		for i := 1; i < BoardSize; i++ {
			target := Coordinates{Row: from.Row + d[0]*i, Col: from.Col + d[1]*i}
			if !target.IsValid() {
				break
			}
			if b.HasFriendly(target, color) {
				break
			}
			candidates = append(candidates, target)
			if b.HasEnemy(target, color) {
				break
			}
		}
	}

	return FilterIllegalMoves(b, from, candidates, color)
}
