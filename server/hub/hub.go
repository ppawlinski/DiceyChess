package hub

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
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
	clients    map[int64]*Client
	mu         sync.RWMutex
	Register   chan *Client
	Unregister chan *Client
}

func New() *Hub {
	return &Hub{
		clients:    make(map[int64]*Client),
		Register:   make(chan *Client),
		Unregister: make(chan *Client),
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
	log.Printf("wiadomość od %s: %s", c.PlayerName, msg.Type)
}
