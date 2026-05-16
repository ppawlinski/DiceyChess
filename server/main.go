package main

import (
	"log"
	"net/http"

	"github.com/ppawlinski/DiceyChess/server/api"
	"github.com/ppawlinski/DiceyChess/server/db"
	"github.com/ppawlinski/DiceyChess/server/game"
	"github.com/ppawlinski/DiceyChess/server/hub"
)

func main() {
	database, err := db.New("chess.db")
	if err != nil {
		log.Fatal(err)
	}

	gm := game.NewGameManager()

	// załaduj trwające gry z bazy
	if err := loadOngoingGames(database, gm); err != nil {
		log.Printf("błąd ładowania gier: %v", err)
	}

	h := hub.New(gm, database)
	go h.Run()

	handler := &api.Handler{DB: database, Hub: h, GameManager: gm}

	http.HandleFunc("/api/login", handler.Login)
	http.HandleFunc("/api/players", handler.GetPlayers)
	http.HandleFunc("/api/games", handler.GetGames)
	http.HandleFunc("/api/games/create", handler.CreateGame)
	http.HandleFunc("/ws", handler.ServeWS)

	log.Println("Serwer działa na :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func loadOngoingGames(database *db.DB, gm *game.GameManager) error {
	games, err := database.GetOngoingGames()
	if err != nil {
		return err
	}
	for _, g := range games {
		if g.State == nil {
			continue
		}
		loaded, err := game.DeserializeGame(*g.State)
		if err != nil {
			log.Printf("błąd deserializacji gry %d: %v", g.ID, err)
			continue
		}
		gm.Add(g.ID, loaded)
	}
	return nil
}
