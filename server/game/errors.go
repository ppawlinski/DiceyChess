package game

import "errors"

var (
	ErrNotYourTurn        = errors.New("not your turn")
	ErrInsufficientBudget = errors.New("insufficient budget")
	ErrIllegalMove        = errors.New("illegal move")
	ErrMustEscapeCheck    = errors.New("must escape check first")
	ErrAlreadyCaptured    = errors.New("piece already captured this turn")
	ErrBoardUnchanged     = errors.New("board must change after turn")
	ErrInvalidPromotion   = errors.New("invalid promotion piece")
	ErrNoPieceAtSource    = errors.New("no piece at source square")
	ErrGameOver           = errors.New("game is already over")
)
