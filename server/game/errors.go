package game

import "errors"

var (
	ErrNotYourTurn        = errors.New("not your turn")
	ErrInsufficientBudget = errors.New("insufficient budget")
	ErrIllegalMove        = errors.New("illegal move")
	ErrMustEscapeCheck    = errors.New("must escape check first")
	ErrAlreadyCaptured    = errors.New("piece already captured this turn")
	ErrWouldRevertBoard   = errors.New("ten ruch cofnąłby planszę do stanu sprzed tury")
	ErrInvalidPromotion   = errors.New("invalid promotion piece")
	ErrNoPieceAtSource    = errors.New("no piece at source square")
	ErrGameOver           = errors.New("game is already over")
	ErrNotAPromotablePawn = errors.New("no promotable pawn at that square")
)
