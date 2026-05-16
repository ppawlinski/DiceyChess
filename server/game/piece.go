package game

type Color int8

const (
	White Color = iota
	Black
	ColorLength
)

type PieceType int8

const (
	PawnType PieceType = iota
	BishopType
	KnightType
	RookType
	QueenType
	KingType
)

type Piece struct {
	Color     Color
	pieceType PieceType
	FirstMove bool
}

func (p *Piece) Type() PieceType {
	return p.pieceType
}

func MoveCost(pt PieceType) int {
	switch pt {
	case PawnType:
		return 1
	case BishopType, KnightType, KingType:
		return 2
	case RookType:
		return 3
	case QueenType:
		return 4
	}
	return 0
}

const CastlingCost = 2
const CheckedKingCost = 1

type Movable interface {
	Move(Coordinates)
	GetPossibleMoves(*Board, Coordinates, Coordinates) []Coordinates
	Type() PieceType
	Piece() *Piece
}

func EffectiveCost(pt PieceType, isKingInCheck bool) int {
	if pt == KingType && isKingInCheck {
		return CheckedKingCost
	}
	return MoveCost(pt)
}

type Pawn struct {
	piece *Piece
}

type Knight struct {
	piece *Piece
}

type Bishop struct {
	piece *Piece
}

type Rook struct {
	piece *Piece
}

type Queen struct {
	piece *Piece
}

type King struct {
	piece *Piece
}

func NewPiece(color Color, pt PieceType) *Piece {
	return &Piece{Color: color, pieceType: pt, FirstMove: true}
}

func NewPawn(color Color) *Pawn {
	return &Pawn{piece: NewPiece(color, PawnType)}
}

func NewKnight(color Color) *Knight {
	return &Knight{piece: NewPiece(color, KnightType)}
}

func NewBishop(color Color) *Bishop {
	return &Bishop{piece: NewPiece(color, BishopType)}
}

func NewRook(color Color) *Rook {
	return &Rook{piece: NewPiece(color, RookType)}
}

func NewQueen(color Color) *Queen {
	return &Queen{piece: NewPiece(color, QueenType)}
}

func NewKing(color Color) *King {
	return &King{piece: NewPiece(color, KingType)}
}

func NewPieceByType(color Color, pt PieceType) Movable {
	switch pt {
	case PawnType:
		return NewPawn(color)
	case KnightType:
		return NewKnight(color)
	case BishopType:
		return NewBishop(color)
	case RookType:
		return NewRook(color)
	case QueenType:
		return NewQueen(color)
	case KingType:
		return NewKing(color)
	}
	return nil
}

func (p *Pawn) Piece() *Piece       { return p.piece }
func (p *Pawn) Type() PieceType     { return PawnType }
func (p *Pawn) Move(to Coordinates) { p.piece.FirstMove = false }

func (k *Knight) Piece() *Piece       { return k.piece }
func (k *Knight) Type() PieceType     { return KnightType }
func (k *Knight) Move(to Coordinates) { k.piece.FirstMove = false }

func (b *Bishop) Piece() *Piece       { return b.piece }
func (b *Bishop) Type() PieceType     { return BishopType }
func (b *Bishop) Move(to Coordinates) { b.piece.FirstMove = false }

func (r *Rook) Piece() *Piece       { return r.piece }
func (r *Rook) Type() PieceType     { return RookType }
func (r *Rook) Move(to Coordinates) { r.piece.FirstMove = false }

func (q *Queen) Piece() *Piece       { return q.piece }
func (q *Queen) Type() PieceType     { return QueenType }
func (q *Queen) Move(to Coordinates) { q.piece.FirstMove = false }

func (k *King) Piece() *Piece       { return k.piece }
func (k *King) Type() PieceType     { return KingType }
func (k *King) Move(to Coordinates) { k.piece.FirstMove = false }

func (p *Pawn) GetPossibleMoves(b *Board, from Coordinates, enPassant Coordinates) []Coordinates {
	return GetPawnMoves(b, from, p.piece.Color, enPassant)
}

func (k *Knight) GetPossibleMoves(b *Board, from Coordinates, enPassant Coordinates) []Coordinates {
	return GetKnightMoves(b, from, k.piece.Color)
}

func (b *Bishop) GetPossibleMoves(board *Board, from Coordinates, enPassant Coordinates) []Coordinates {
	return GetBishopMoves(board, from, b.piece.Color)
}

func (r *Rook) GetPossibleMoves(b *Board, from Coordinates, enPassant Coordinates) []Coordinates {
	return GetRookMoves(b, from, r.piece.Color)
}

func (q *Queen) GetPossibleMoves(b *Board, from Coordinates, enPassant Coordinates) []Coordinates {
	return GetQueenMoves(b, from, q.piece.Color)
}

func (k *King) GetPossibleMoves(b *Board, from Coordinates, enPassant Coordinates) []Coordinates {
	return append(GetKingMoves(b, from, k.piece.Color), GetCastlingMoves(b, from, k.piece.Color)...)
}
