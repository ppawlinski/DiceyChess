package db

import (
	"database/sql"

	"github.com/ppawlinski/DiceyChess/server/models"
	_ "modernc.org/sqlite"
)

type DB struct {
	conn *sql.DB
}

func New(path string) (*DB, error) {
	conn, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}

	if err := conn.Ping(); err != nil {
		return nil, err
	}

	d := &DB{conn: conn}

	if err := d.migrate(); err != nil {
		return nil, err
	}

	return d, nil
}

func (d *DB) migrate() error {
	_, err := d.conn.Exec(`
		CREATE TABLE IF NOT EXISTS players (
			id         INTEGER PRIMARY KEY,
			name       TEXT UNIQUE NOT NULL,
			token      TEXT UNIQUE NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);

		CREATE TABLE IF NOT EXISTS games (
			id          INTEGER PRIMARY KEY,
			white_id    INTEGER REFERENCES players(id),
			black_id    INTEGER REFERENCES players(id),
			status      TEXT NOT NULL DEFAULT 'ongoing',
			result      TEXT,
			pgn         TEXT,
			state       TEXT,
			created_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
			finished_at DATETIME
		);
	`)
	return err
}

func (d *DB) GetOrCreatePlayer(name, token string) (*models.Player, error) {
	row := d.conn.QueryRow(`SELECT id, name, token, created_at FROM players WHERE name = ?`, name)

	var p models.Player
	err := row.Scan(&p.ID, &p.Name, &p.Token, &p.CreatedAt)
	if err == sql.ErrNoRows {
		_, err := d.conn.Exec(`INSERT INTO players (name, token) VALUES (?, ?)`, name, token)
		if err != nil {
			return nil, err
		}
		// pobierz z bazy żeby mieć created_at
		row = d.conn.QueryRow(`SELECT id, name, token, created_at FROM players WHERE name = ?`, name)
		if err := row.Scan(&p.ID, &p.Name, &p.Token, &p.CreatedAt); err != nil {
			return nil, err
		}
		return &p, nil
	}
	if err != nil {
		return nil, err
	}

	return &p, nil
}

func (d *DB) GetPlayerByToken(token string) (*models.Player, error) {
	row := d.conn.QueryRow(`SELECT id, name, token, created_at FROM players WHERE token = ?`, token)

	var p models.Player
	err := row.Scan(&p.ID, &p.Name, &p.Token, &p.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &p, nil
}

func (d *DB) GetAllPlayers() ([]models.Player, error) {
	rows, err := d.conn.Query(`SELECT id, name, token, created_at FROM players ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var players []models.Player
	for rows.Next() {
		var p models.Player
		if err := rows.Scan(&p.ID, &p.Name, &p.Token, &p.CreatedAt); err != nil {
			return nil, err
		}
		players = append(players, p)
	}
	return players, nil
}

func (d *DB) CreateGame(whiteID, blackID int64, state string) (*models.Game, error) {
	res, err := d.conn.Exec(`
		INSERT INTO games (white_id, black_id, status, state)
		VALUES (?, ?, 'ongoing', ?)
	`, whiteID, blackID, state)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return d.GetGame(id)
}

func (d *DB) GetGame(id int64) (*models.Game, error) {
	row := d.conn.QueryRow(`
		SELECT id, white_id, black_id, status, result, pgn, state, created_at, finished_at
		FROM games WHERE id = ?
	`, id)

	var g models.Game
	err := row.Scan(&g.ID, &g.WhiteID, &g.BlackID, &g.Status, &g.Result, &g.PGN, &g.State, &g.CreatedAt, &g.FinishedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &g, nil
}

func (d *DB) GetOngoingGames() ([]models.Game, error) {
	rows, err := d.conn.Query(`
		SELECT id, white_id, black_id, status, result, pgn, state, created_at, finished_at
		FROM games WHERE status = 'ongoing'
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var games []models.Game
	for rows.Next() {
		var g models.Game
		if err := rows.Scan(&g.ID, &g.WhiteID, &g.BlackID, &g.Status, &g.Result, &g.PGN, &g.State, &g.CreatedAt, &g.FinishedAt); err != nil {
			return nil, err
		}
		games = append(games, g)
	}
	return games, nil
}

func (d *DB) UpdateGameState(id int64, state string) error {
	_, err := d.conn.Exec(`UPDATE games SET state = ? WHERE id = ?`, state, id)
	return err
}

func (d *DB) FinishGame(id int64, result string) error {
	_, err := d.conn.Exec(`
		UPDATE games SET status = 'finished', result = ?, finished_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, result, id)
	return err
}
