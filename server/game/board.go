package game

type Board struct {
	fields       [BoardSize][BoardSize]Movable
	kingPosition [ColorLength]Coordinates
}

func NewBoard() *Board {
	b := &Board{}
	b.kingPosition[White] = InvalidCoordinates
	b.kingPosition[Black] = InvalidCoordinates
	return b
}

func (b *Board) Get(c Coordinates) Movable {
	if !c.IsValid() {
		return nil
	}
	return b.fields[c.Row][c.Col]
}

func (b *Board) Set(c Coordinates, m Movable) {
	if !c.IsValid() {
		return
	}
	b.fields[c.Row][c.Col] = m
	if m != nil && m.Type() == KingType {
		b.kingPosition[m.Piece().Color] = c
	}
}

func (b *Board) Remove(c Coordinates) {
	if !c.IsValid() {
		return
	}
	b.fields[c.Row][c.Col] = nil
}

func (b *Board) KingPosition(color Color) Coordinates {
	return b.kingPosition[color]
}

func (b *Board) IsEmpty(c Coordinates) bool {
	return b.Get(c) == nil
}

func (b *Board) HasEnemy(c Coordinates, color Color) bool {
	m := b.Get(c)
	return m != nil && m.Piece().Color != color
}

func (b *Board) HasFriendly(c Coordinates, color Color) bool {
	m := b.Get(c)
	return m != nil && m.Piece().Color == color
}

func (b *Board) Clone() *Board {
	clone := &Board{}
	clone.kingPosition = b.kingPosition
	for row := range b.fields {
		for col := range b.fields[row] {
			clone.fields[row][col] = b.fields[row][col]
		}
	}
	return clone
}

func (b *Board) Reset() {
	b.Set(Coordinates{0, 0}, NewRook(Black))
	b.Set(Coordinates{0, 1}, NewKnight(Black))
	b.Set(Coordinates{0, 2}, NewBishop(Black))
	b.Set(Coordinates{0, 3}, NewKing(Black))
	b.Set(Coordinates{0, 4}, NewQueen(Black))
	b.Set(Coordinates{0, 5}, NewBishop(Black))
	b.Set(Coordinates{0, 6}, NewKnight(Black))
	b.Set(Coordinates{0, 7}, NewRook(Black))

	b.Set(Coordinates{7, 0}, NewRook(White))
	b.Set(Coordinates{7, 1}, NewKnight(White))
	b.Set(Coordinates{7, 2}, NewBishop(White))
	b.Set(Coordinates{7, 3}, NewKing(White))
	b.Set(Coordinates{7, 4}, NewQueen(White))
	b.Set(Coordinates{7, 5}, NewBishop(White))
	b.Set(Coordinates{7, 6}, NewKnight(White))
	b.Set(Coordinates{7, 7}, NewRook(White))

	for col := 0; col < BoardSize; col++ {
		b.Set(Coordinates{1, col}, NewPawn(Black))
		b.Set(Coordinates{6, col}, NewPawn(White))
	}
}
