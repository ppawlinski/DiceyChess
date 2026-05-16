package main

import (
	"log"
	"net/http"

	"github.com/ppawlinski/DiceyChess/server/api"
	"github.com/ppawlinski/DiceyChess/server/db"
	"github.com/ppawlinski/DiceyChess/server/hub"
)

func main() {
	database, err := db.New("chess.db")
	if err != nil {
		log.Fatal(err)
	}

	h := hub.New()
	go h.Run()

	handler := &api.Handler{DB: database, Hub: h}

	http.HandleFunc("/api/login", handler.Login)
	http.HandleFunc("/api/players", handler.GetPlayers)
	http.HandleFunc("/ws", handler.ServeWS)
	http.HandleFunc("/api/games", handler.GetGames)
	http.HandleFunc("/api/games/create", handler.CreateGame)

	log.Println("Serwer działa na :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
