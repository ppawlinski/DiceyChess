package main

import (
	"fmt"

	"github.com/ppawlinski/DiceyChess/assets"
	"github.com/ppawlinski/DiceyChess/config"

	"github.com/hajimehoshi/ebiten/v2"
)

const (
	maxDimensions = 8
	tileOffset    = 1
)

type Draggable interface {
	SetDragOffset(x, y int)
}

type Coordinates struct {
	col int
	row int
}

func (c *Coordinates) Undefined() bool {
	return c.col == -1 && c.row == -1
}

func (c *Coordinates) Reset() {
	c.col = -1
	c.row = -1
}

func (c *Coordinates) Valid() bool {
	return c.col >= 0 && c.col < maxDimensions && c.row >= 0 && c.row < maxDimensions
}

func (c *Coordinates) Equal(col, row int) bool {
	return c.col == col && c.row == row
}

type Movable interface {
	Move(Coordinates)
	GetPossibleMoves(*Board, Direction, Coordinates) []Coordinates
	Type() PieceType
	Piece() *Piece
	Cost(*GameState) int
}

type Drawable interface {
	Draw(*ebiten.Image, int, int)
}

type MovableDrawable interface {
	Movable
	Drawable
}

type Board struct {
	fields       [maxDimensions][maxDimensions]MovableDrawable
	kingPosition [ColorLength]Coordinates
	direction    Direction
}

func NewBoard() *Board {
	return &Board{
		kingPosition: [2]Coordinates{{1, 1}, {1, 1}},
		direction:    Normal,
	}
}

func (b *Board) Draw(screen *ebiten.Image, state *GameState) {
	var rect *ebiten.Image
	for row := range maxDimensions {
		for col := range maxDimensions {
			if (row%2 == 1 && col%2 == 1) || (row%2 == 0 && col%2 == 0) {
				rect = assets.Images.DarkSquare
			} else {
				rect = assets.Images.LightSquare
			}
			for i := range state.possibleMoves {
				currentCoordinates := Coordinates{col, row}
				if state.possibleMoves[i] == currentCoordinates {
					rect = assets.Images.HighlightedSquare
				}
			}
			x := col * (config.TileSize + tileOffset)
			y := row * (config.TileSize + tileOffset)
			drawOptions := ebiten.DrawImageOptions{}
			drawOptions.GeoM.Translate(float64(x), float64(y))
			screen.DrawImage(rect, &drawOptions)
		}
	}

	for row := range maxDimensions {
		for col := range maxDimensions {
			x := col * (config.TileSize + tileOffset)
			y := row * (config.TileSize + tileOffset)
			piece := b.fields[col][row]
			if piece != nil {
				piece.Draw(screen, x, y)
			}
		}
	}
}

func (b *Board) Reset() {
	//!!!3PPA todo change direction (orientation)
	b.fields[0][0] = &Rook{NewPiece(Black)}
	b.fields[1][0] = &Knight{NewPiece(Black)}
	b.fields[2][0] = &Bishop{NewPiece(Black)}
	b.fields[3][0] = &King{NewPiece(Black)}
	b.fields[4][0] = &Queen{NewPiece(Black)}
	b.fields[5][0] = &Bishop{NewPiece(Black)}
	b.fields[6][0] = &Knight{NewPiece(Black)}
	b.fields[7][0] = &Rook{NewPiece(Black)}

	b.fields[0][7] = &Rook{NewPiece(White)}
	b.fields[1][7] = &Knight{NewPiece(White)}
	b.fields[2][7] = &Bishop{NewPiece(White)}
	b.fields[3][7] = &King{NewPiece(White)}
	b.fields[4][7] = &Queen{NewPiece(White)}
	b.fields[5][7] = &Bishop{NewPiece(White)}
	b.fields[6][7] = &Knight{NewPiece(White)}
	b.fields[7][7] = &Rook{NewPiece(White)}

	b.fields[0][1] = NewPawn(Black)
	b.fields[1][1] = NewPawn(Black)
	b.fields[2][1] = NewPawn(Black)
	b.fields[3][1] = NewPawn(Black)
	b.fields[4][1] = NewPawn(Black)
	b.fields[5][1] = NewPawn(Black)
	b.fields[6][1] = NewPawn(Black)
	b.fields[7][1] = NewPawn(Black)
	b.fields[0][6] = NewPawn(White)
	b.fields[1][6] = NewPawn(White)
	b.fields[2][6] = NewPawn(White)
	b.fields[3][6] = NewPawn(White)
	b.fields[4][6] = NewPawn(White)
	b.fields[5][6] = NewPawn(White)
	b.fields[6][6] = NewPawn(White)
	b.fields[7][6] = NewPawn(White)

	b.kingPosition = [2]Coordinates{{3, 7}, {3, 0}}
}

func (b *Board) HitCheck(state *GameState) Coordinates {
	resultCol := -1
	resultRow := -1
	x, y := ebiten.CursorPosition()
	col := x / (config.TileSize + tileOffset)
	row := y / (config.TileSize + tileOffset)
	if x > 0 && y > 0 && col >= 0 && col < maxDimensions && row >= 0 && row < maxDimensions {
		resultCol = col
		resultRow = row
	}

	return Coordinates{resultCol, resultRow}
}

func (b *Board) SelectPiece(s *GameState) {
	selectedPiece := Coordinates{-1, -1}
	field := b.HitCheck(s)
	fmt.Println("selected", field)
	fmt.Println(s.colorToMove)
	if b.GetPiece(field) != nil && b.GetPiece(field).Piece().color == s.colorToMove {
		selectedPiece = field
	}
	s.selectedPiece = selectedPiece
}

func (b *Board) GetPossibleMoves(c Coordinates) []Coordinates {
	return b.fields[c.col][c.row].GetPossibleMoves(b, Normal, Coordinates{c.col, c.row})
}

func (b *Board) MoveSelected(state *GameState, selectedMove Coordinates) {
	dropped := b.fields[state.selectedPiece.col][state.selectedPiece.row]
	dropped.Piece().SetDragOffset(0, 0)

	if dropped.Cost(state) > state.currentBudget {
		state.possibleMoves = nil
		state.selectedPiece.Reset()
	}

	moves := state.possibleMoves
	for _, move := range moves {
		if move.col == selectedMove.col && move.row == selectedMove.row {
			state.possibleMoves = nil
			handleEnPassant(dropped, state, move, b)

			dropped.Piece().firstMove = false
			b.fields[move.col][move.row] = b.fields[state.selectedPiece.col][state.selectedPiece.row]
			if b.fields[move.col][move.row].Type() == KingType {
				b.kingPosition[b.fields[move.col][move.row].Piece().color] = move
			}
			b.fields[state.selectedPiece.col][state.selectedPiece.row] = nil
			state.currentBudget -= dropped.Cost(state)
			state.selectedPiece.Reset()
			break
		}
	}
}

func handleEnPassant(dropped MovableDrawable, state *GameState, move Coordinates, b *Board) {
	if pawn, ok := dropped.(EnPassantable); ok {
		if state.selectedPiece.row-move.row == 2 || state.selectedPiece.row-move.row == -2 {
			if state.enPassant.Valid() {
				b.fields[state.enPassant.col][state.enPassant.row].(EnPassantable).SetEnPassantable(false)
			}
			state.enPassant = move
			pawn.SetEnPassantable(true)
		} else {
			//en passant taking
			if move.col != state.selectedPiece.col && state.enPassant.Equal(move.col, state.selectedPiece.row) {
				b.fields[move.col][state.selectedPiece.row] = nil
			} else {
				if state.enPassant.Valid() {
					b.fields[state.enPassant.col][state.enPassant.row].(EnPassantable).SetEnPassantable(false)
				}
			}
			state.enPassant.Reset()
		}
	} else {
		if state.enPassant.Valid() {
			b.fields[state.enPassant.col][state.enPassant.row].(EnPassantable).SetEnPassantable(false)
		}
		state.enPassant.Reset()
	}
}

func (b *Board) GetPiece(position Coordinates) MovableDrawable {
	if position.Valid() {
		return b.fields[position.col][position.row]
	} else {
		return nil
	}
}

func (b *Board) GetColor(position Coordinates) Color {
	var color Color

	if position.Valid() && b.fields[position.col][position.row] != nil {
		color = b.fields[position.col][position.row].Piece().color
	}

	return color
}
