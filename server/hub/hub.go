package hub

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/ppawlinski/DiceyChess/server/db"
	"github.com/ppawlinski/DiceyChess/server/game"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // akceptuj każdy origin
	},
}

type Message struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

type Client struct {
	PlayerID   int64
	PlayerName string
	conn       *websocket.Conn
	send       chan Message
}

type Hub struct {
	clients     map[int64]*Client
	gameClients map[int64][]int64 // gameID → []playerID
	mu          sync.RWMutex
	Register    chan *Client
	Unregister  chan *Client
	GameManager *game.GameManager
	DB          *db.DB
}

func New(gm *game.GameManager, database *db.DB) *Hub {
	return &Hub{
		clients:     make(map[int64]*Client),
		gameClients: make(map[int64][]int64),
		Register:    make(chan *Client),
		Unregister:  make(chan *Client),
		GameManager: gm,
		DB:          database,
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.Register:
			h.mu.Lock()
			h.clients[client.PlayerID] = client
			h.mu.Unlock()
			log.Printf("%s połączony", client.PlayerName)
			h.broadcastOnlinePlayers()

		case client := <-h.Unregister:
			h.mu.Lock()
			delete(h.clients, client.PlayerID)
			h.mu.Unlock()
			log.Printf("%s rozłączony", client.PlayerName)
			h.broadcastOnlinePlayers()
		}
	}
}

func (h *Hub) broadcastOnlinePlayers() {
	h.mu.RLock()
	defer h.mu.RUnlock()

	type playerInfo struct {
		ID   int64  `json:"id"`
		Name string `json:"name"`
	}

	players := []playerInfo{}
	for _, c := range h.clients {
		players = append(players, playerInfo{ID: c.PlayerID, Name: c.PlayerName})
	}

	payload, _ := json.Marshal(players)
	msg := Message{
		Type:    "players_online",
		Payload: payload,
	}

	for _, c := range h.clients {
		c.send <- msg
	}
}

func (h *Hub) SendTo(playerID int64, msg Message) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()

	client, ok := h.clients[playerID]
	if !ok {
		return false
	}
	client.send <- msg
	return true
}

func (h *Hub) IsOnline(playerID int64) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	_, ok := h.clients[playerID]
	return ok
}

func NewClient(playerID int64, playerName string, conn *websocket.Conn) *Client {
	return &Client{
		PlayerID:   playerID,
		PlayerName: playerName,
		conn:       conn,
		send:       make(chan Message, 32),
	}
}

func (c *Client) WritePump() {
	for msg := range c.send {
		if err := c.conn.WriteJSON(msg); err != nil {
			log.Printf("błąd wysyłania do %s: %v", c.PlayerName, err)
			return
		}
	}
}

func (c *Client) ReadPump(h *Hub, onMessage func(*Client, Message)) {
	defer func() {
		h.Unregister <- c
		c.conn.Close()
	}()

	for {
		var msg Message
		if err := c.conn.ReadJSON(&msg); err != nil {
			break
		}
		onMessage(c, msg)
	}
}

func (h *Hub) ServeWS(w http.ResponseWriter, r *http.Request, playerID int64, playerName string) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("błąd upgrade: %v", err)
		return
	}

	client := NewClient(playerID, playerName, conn)
	h.Register <- client

	go client.WritePump()
	client.ReadPump(h, h.handleMessage)
}

func (h *Hub) handleMessage(c *Client, msg Message) {
	switch msg.Type {
	case "get_legal_moves":
		h.handleGetLegalMoves(c, msg.Payload)
	case "make_move":
		h.handleMakeMove(c, msg.Payload)
	case "promote_pawn":
		h.handlePromotePawn(c, msg.Payload)
	case "end_turn":
		h.handleEndTurn(c, msg.Payload)
	case "join_game":
		h.handleJoinGame(c, msg.Payload)
	case "roll_dice":
		h.handleRollDice(c, msg.Payload)
	case "get_history":
		h.handleGetHistory(c, msg.Payload)
	}
}

func (h *Hub) handleRollDice(c *Client, payload json.RawMessage) {
	var req struct {
		GameID int64 `json:"game_id"`
	}
	if err := json.Unmarshal(payload, &req); err != nil {
		h.sendError(c, "invalid payload")
		return
	}

	g, ok := h.GameManager.Get(req.GameID)
	if !ok {
		h.sendError(c, "game not found")
		return
	}

	if g.State.TurnStarted {
		h.sendError(c, "turn already started")
		return
	}

	currentPlayerID := g.WhiteID
	if g.State.ColorToMove == game.Black {
		currentPlayerID = g.BlackID
	}
	if c.PlayerID != currentPlayerID {
		h.sendError(c, "not your turn")
		return
	}

	g.StartTurn()

	state, pgn, _ := g.SerializeForDB()
	h.DB.UpdateGame(req.GameID, state, pgn)

	h.broadcastGameState(req.GameID, g, "dice_roll")
}

func (h *Hub) handleGetLegalMoves(c *Client, payload json.RawMessage) {
	var req struct {
		GameID int64            `json:"game_id"`
		From   game.Coordinates `json:"from"`
	}
	if err := json.Unmarshal(payload, &req); err != nil {
		h.sendError(c, "invalid payload")
		return
	}

	g, ok := h.GameManager.Get(req.GameID)
	if !ok {
		h.sendError(c, "game not found")
		return
	}

	moves, err := g.GetLegalMoves(req.From)
	if err != nil {
		h.sendError(c, err.Error())
		return
	}

	responsePayload, _ := json.Marshal(map[string]any{
		"game_id": req.GameID,
		"from":    req.From,
		"moves":   moves,
	})
	c.send <- Message{Type: "legal_moves", Payload: responsePayload}
}

func (h *Hub) handleMakeMove(c *Client, payload json.RawMessage) {
	var req struct {
		GameID int64            `json:"game_id"`
		From   game.Coordinates `json:"from"`
		To     game.Coordinates `json:"to"`
	}
	if err := json.Unmarshal(payload, &req); err != nil {
		h.sendError(c, "invalid payload")
		return
	}

	g, ok := h.GameManager.Get(req.GameID)
	if !ok {
		h.sendError(c, "game not found")
		return
	}

	err := g.MakeMove(game.MoveRequest{
		From: req.From,
		To:   req.To,
	})
	if err != nil {
		h.sendError(c, err.Error())
		return
	}

	// zapisz stan do bazy
	state, pgn, err := g.SerializeForDB()
	if err == nil {
		h.DB.UpdateGame(req.GameID, state, pgn)
	}

	// jeśli gra skończona
	if g.State.IsOver {
		result := "draw"
		if g.State.Winner != nil {
			if *g.State.Winner == game.White {
				result = "white"
			} else {
				result = "black"
			}
		}
		h.DB.FinishGame(req.GameID, result)
		h.GameManager.Remove(req.GameID)
	}

	// wyślij nowy stan obu graczom
	h.broadcastGameState(req.GameID, g, "game_state")
}

func (h *Hub) handlePromotePawn(c *Client, payload json.RawMessage) {
	var req struct {
		GameID    int64            `json:"game_id"`
		At        game.Coordinates `json:"at"`
		PromoteTo game.PieceType   `json:"promote_to"`
	}
	if err := json.Unmarshal(payload, &req); err != nil {
		h.sendError(c, "invalid payload")
		return
	}

	g, ok := h.GameManager.Get(req.GameID)
	if !ok {
		h.sendError(c, "game not found")
		return
	}

	currentPlayerID := g.WhiteID
	if g.State.ColorToMove == game.Black {
		currentPlayerID = g.BlackID
	}
	if c.PlayerID != currentPlayerID {
		h.sendError(c, "not your turn")
		return
	}

	if err := g.Promote(game.PromoteRequest{At: req.At, PromoteTo: req.PromoteTo}); err != nil {
		h.sendError(c, err.Error())
		return
	}

	state, pgn, err := g.SerializeForDB()
	if err == nil {
		h.DB.UpdateGame(req.GameID, state, pgn)
	}

	if g.State.IsOver {
		result := "draw"
		if g.State.Winner != nil {
			if *g.State.Winner == game.White {
				result = "white"
			} else {
				result = "black"
			}
		}
		h.DB.FinishGame(req.GameID, result)
		h.GameManager.Remove(req.GameID)
	}

	h.broadcastGameState(req.GameID, g, "game_state")
}

func (h *Hub) handleEndTurn(c *Client, payload json.RawMessage) {
	var req struct {
		GameID int64 `json:"game_id"`
	}
	if err := json.Unmarshal(payload, &req); err != nil {
		h.sendError(c, "invalid payload")
		return
	}

	g, ok := h.GameManager.Get(req.GameID)
	if !ok {
		h.sendError(c, "game not found")
		return
	}

	// sprawdź czy to tura tego gracza
	currentPlayerID := g.WhiteID
	if g.State.ColorToMove == game.Black {
		currentPlayerID = g.BlackID
	}
	if c.PlayerID != currentPlayerID {
		h.sendError(c, "not your turn")
		return
	}

	if g.State.TurnStartBoardHash != "" && g.Board.Hash() == g.State.TurnStartBoardHash && g.CanAffordToMove() {
		h.sendError(c, "plansza musi zmienić się przed zakończeniem tury")
		return
	}

	if err := g.EndTurn(); err != nil {
		h.sendError(c, err.Error())
		return
	}

	state, pgn, _ := g.SerializeForDB()
	h.DB.UpdateGame(req.GameID, state, pgn)

	h.broadcastGameState(req.GameID, g, "game_state")
}

func (h *Hub) handleGetHistory(c *Client, rawPayload json.RawMessage) {
	var req struct {
		GameID int64 `json:"game_id"`
	}
	if err := json.Unmarshal(rawPayload, &req); err != nil {
		h.sendError(c, "invalid payload")
		return
	}

	var history game.History

	g, ok := h.GameManager.Get(req.GameID)
	if ok {
		history = g.History
	} else {
		// Finished game — load from DB
		dbGame, err := h.DB.GetGame(req.GameID)
		if err != nil || dbGame == nil || dbGame.State == nil {
			h.sendError(c, "game not found")
			return
		}
		loaded, err := game.DeserializeGame(*dbGame.State)
		if err != nil {
			h.sendError(c, "failed to load game")
			return
		}
		history = loaded.History
	}

	responsePayload, _ := json.Marshal(map[string]any{
		"game_id": req.GameID,
		"turns":   history,
	})
	c.send <- Message{Type: "history", Payload: responsePayload}
}

func (h *Hub) handleJoinGame(c *Client, payload json.RawMessage) {
	var req struct {
		GameID int64 `json:"game_id"`
	}
	if err := json.Unmarshal(payload, &req); err != nil {
		h.sendError(c, "invalid payload")
		return
	}

	g, ok := h.GameManager.Get(req.GameID)
	if ok {
		// Ongoing game — add to gameClients so future updates reach this client
		h.JoinGame(req.GameID, c.PlayerID)
		h.broadcastGameState(req.GameID, g, "game_state")
		return
	}

	// Finished game — load from DB and send directly; no live updates needed
	dbGame, err := h.DB.GetGame(req.GameID)
	if err != nil || dbGame == nil || dbGame.State == nil {
		h.sendError(c, "game not found")
		return
	}
	loaded, err := game.DeserializeGame(*dbGame.State)
	if err != nil {
		h.sendError(c, "failed to load game")
		return
	}
	msg := buildGameStateMsg(req.GameID, loaded, "game_state")
	c.send <- msg
}

func buildGameStateMsg(gameID int64, g *game.Game, messageType string) Message {
	inCheck := game.KingInCheck(g.Board, g.State.ColorToMove)

	type moveCoords struct {
		From game.Coordinates `json:"from"`
		To   game.Coordinates `json:"to"`
	}
	var lastMove *moveCoords
	if len(g.History) > 0 {
		lastHT := g.History[len(g.History)-1]
		if len(lastHT.Moves) > 0 {
			m := lastHT.Moves[len(lastHT.Moves)-1]
			lastMove = &moveCoords{From: m.From, To: m.To}
		}
	}

	payload, _ := json.Marshal(map[string]any{
		"game_id":   gameID,
		"board":     g.Board,
		"state":     g.State,
		"white_id":  g.WhiteID,
		"black_id":  g.BlackID,
		"in_check":  inCheck,
		"last_move": lastMove,
	})
	return Message{Type: messageType, Payload: payload}
}

func (h *Hub) broadcastGameState(gameID int64, g *game.Game, messageType string) {
	h.mu.RLock()
	players := h.gameClients[gameID]
	h.mu.RUnlock()

	msg := buildGameStateMsg(gameID, g, messageType)
	for _, playerID := range players {
		h.SendTo(playerID, msg)
	}
}

func (h *Hub) JoinGame(gameID, playerID int64) {
	h.mu.Lock()
	defer h.mu.Unlock()

	for _, id := range h.gameClients[gameID] {
		if id == playerID {
			return
		}
	}

	h.gameClients[gameID] = append(h.gameClients[gameID], playerID)
}

func (h *Hub) LeaveGame(gameID, playerID int64) {
	h.mu.Lock()
	defer h.mu.Unlock()
	players := h.gameClients[gameID]
	for i, id := range players {
		if id == playerID {
			h.gameClients[gameID] = append(players[:i], players[i+1:]...)
			break
		}
	}
}

func (h *Hub) sendError(c *Client, msg string) {
	payload, _ := json.Marshal(map[string]string{"error": msg})
	c.send <- Message{Type: "error", Payload: payload}
}
