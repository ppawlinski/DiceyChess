package game

const BoardSize = 8

type Coordinates struct {
	Row int
	Col int
}

func (c Coordinates) IsValid() bool {
	return c.Row >= 0 && c.Row < BoardSize && c.Col >= 0 && c.Col < BoardSize
}

func (c Coordinates) Equals(other Coordinates) bool {
	return c.Row == other.Row && c.Col == other.Col
}

var InvalidCoordinates = Coordinates{-1, -1}
