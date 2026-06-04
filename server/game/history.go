package game

import (
	"encoding/json"
	"fmt"
	"strings"
)

// MoveRecord stores notation and board snapshot for one move.
type MoveRecord struct {
	Notation  string      `json:"notation"`
	BoardJSON string      `json:"board"`
	From      Coordinates `json:"from"`
	To        Coordinates `json:"to"`
}

// HalfTurn holds all moves by one player in a single turn.
type HalfTurn struct {
	Num   int          `json:"num"`   // pair number: increments when White starts
	Color Color        `json:"color"` // 0=White, 1=Black
	Roll  int          `json:"roll"`
	Moves []MoveRecord `json:"moves"`
}

// History is the ordered list of half-turns for a game.
type History []HalfTurn

// ToPGN renders a human-readable string close to standard PGN.
// Example:
//
//	1. [5] e4 Nf3  [3] e5
//	2. [2] Bb5  [6] Nc6 Nf6
func (h History) ToPGN() string {
	var sb strings.Builder
	for _, ht := range h {
		if ht.Color == White {
			sb.WriteString(fmt.Sprintf("%d. [%d]", ht.Num, ht.Roll))
		} else {
			sb.WriteString(fmt.Sprintf("  [%d]", ht.Roll))
		}
		for _, m := range ht.Moves {
			sb.WriteByte(' ')
			sb.WriteString(m.Notation)
		}
		if ht.Color == Black {
			sb.WriteByte('\n')
		}
	}
	return strings.TrimSpace(sb.String())
}

// MoveToAlgebraic converts a move to standard algebraic notation (SAN).
func MoveToAlgebraic(pt PieceType, from, to Coordinates, isCapture, isCheck, isCheckmate bool) string {
	// Castling
	if pt == KingType && abs(to.Col-from.Col) == 2 {
		s := "O-O"
		if to.Col < from.Col {
			s = "O-O-O"
		}
		return s + checkSuffix(isCheck, isCheckmate)
	}

	var sb strings.Builder
	switch pt {
	case KnightType:
		sb.WriteByte('N')
	case BishopType:
		sb.WriteByte('B')
	case RookType:
		sb.WriteByte('R')
	case QueenType:
		sb.WriteByte('Q')
	case KingType:
		sb.WriteByte('K')
	// PawnType: no prefix
	}

	// Pawn capture: include departure file
	if pt == PawnType && isCapture {
		sb.WriteByte(byte('a' + from.Col))
	}
	if isCapture {
		sb.WriteByte('x')
	}

	// Destination square
	sb.WriteByte(byte('a' + to.Col))
	sb.WriteByte(byte('0' + (8 - to.Row)))
	sb.WriteString(checkSuffix(isCheck, isCheckmate))
	return sb.String()
}

// PromotionSuffix returns the =X suffix for the promoted piece type.
func PromotionSuffix(pt PieceType) string {
	switch pt {
	case QueenType:
		return "=Q"
	case RookType:
		return "=R"
	case BishopType:
		return "=B"
	case KnightType:
		return "=N"
	}
	return ""
}

func checkSuffix(isCheck, isCheckmate bool) string {
	if isCheckmate {
		return "#"
	}
	if isCheck {
		return "+"
	}
	return ""
}

func boardToJSON(b *Board) string {
	data, _ := json.Marshal(b)
	return string(data)
}
