package models

import "time"

type Player struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Token     string    `json:"-"` // "-" oznacza: nie serializuj do JSON
	CreatedAt time.Time `json:"created_at"`
}

type Game struct {
	ID         int64      `json:"id"`
	WhiteID    int64      `json:"white_id"`
	BlackID    int64      `json:"black_id"`
	Status     string     `json:"status"`
	Result     *string    `json:"result"`
	PGN        *string    `json:"pgn"`
	State      *string    `json:"-"`
	CreatedAt  time.Time  `json:"created_at"`
	FinishedAt *time.Time `json:"finished_at"`
}
