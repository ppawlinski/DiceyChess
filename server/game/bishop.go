package game

func GetBishopMoves(b *Board, from Coordinates, color Color) []Coordinates {
	var moves []Coordinates

	directions := [4][2]int{{1, 1}, {1, -1}, {-1, 1}, {-1, -1}}
	for _, d := range directions {
		for i := 1; i < BoardSize; i++ {
			target := Coordinates{Row: from.Row + d[0]*i, Col: from.Col + d[1]*i}
			if !target.IsValid() {
				break
			}
			if b.HasFriendly(target, color) {
				break
			}
			moves = append(moves, target)
			if b.HasEnemy(target, color) {
				break
			}
		}
	}

	return FilterIllegalMoves(b, from, moves, color)
}
