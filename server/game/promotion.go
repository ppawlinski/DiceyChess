package game

func CanPromoteTo(pt PieceType) bool {
	switch pt {
	case QueenType, RookType, BishopType, KnightType:
		return true
	}
	return false
}

func IsPromotionSquare(c Coordinates, color Color) bool {
	if color == White {
		return c.Row == 0
	}
	return c.Row == BoardSize-1
}

func HandlePromotion(b *Board, to Coordinates, color Color, promoteTo PieceType) {
	b.Set(to, NewPieceByType(color, promoteTo))
}
