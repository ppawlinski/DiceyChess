package api

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"time"

	"github.com/ppawlinski/DiceyChess/server/db"
	"github.com/ppawlinski/DiceyChess/server/game"
	"github.com/ppawlinski/DiceyChess/server/hub"
	"github.com/ppawlinski/DiceyChess/server/models"
)

type Handler struct {
	DB          *db.DB
	Hub         *hub.Hub
	GameManager *game.GameManager
}

func generateToken() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid name"})
		return
	}

	token, err := generateToken()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "server error"})
		return
	}

	player, err := h.DB.GetOrCreatePlayer(req.Name, token)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "server error"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"token":  player.Token,
		"player": player,
	})
}

func (h *Handler) GetPlayers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	players, err := h.DB.GetAllPlayers()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "server error"})
		return
	}

	if players == nil {
		players = []models.Player{}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"players": players,
	})
}

func (h *Handler) ServeWS(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		http.Error(w, "brak tokenu", http.StatusUnauthorized)
		return
	}

	player, err := h.DB.GetPlayerByToken(token)
	if err != nil || player == nil {
		http.Error(w, "nieznany token", http.StatusUnauthorized)
		return
	}

	h.Hub.ServeWS(w, r, player.ID, player.Name)
}

func (h *Handler) CreateGame(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	// autoryzacja przez token
	token := r.Header.Get("Authorization")
	if token == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "brak tokenu"})
		return
	}

	caller, err := h.DB.GetPlayerByToken(token)
	if err != nil || caller == nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "nieznany token"})
		return
	}

	var req struct {
		OpponentID int64 `json:"opponent_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.OpponentID == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "brak opponent_id"})
		return
	}

	if req.OpponentID == caller.ID {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "nie możesz grać sam ze sobą"})
		return
	}

	// losowanie kolorów
	whiteID, blackID := caller.ID, req.OpponentID
	if time.Now().UnixNano()%2 == 0 {
		whiteID, blackID = blackID, whiteID
	}

	// stwórz grę z początkowym stanem
	seed := time.Now().UnixNano()
	newGame := game.NewGame(seed)
	state, err := newGame.Serialize()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "server error"})
		return
	}

	dbGame, err := h.DB.CreateGame(whiteID, blackID, state)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "server error"})
		return
	}

	h.GameManager.Add(dbGame.ID, newGame)

	// powiadom przeciwnika przez WebSocket jeśli online
	payload, _ := json.Marshal(dbGame)
	h.Hub.SendTo(req.OpponentID, hub.Message{
		Type:    "game_started",
		Payload: payload,
	})

	writeJSON(w, http.StatusOK, map[string]any{
		"game": dbGame,
	})
}

func (h *Handler) GetGames(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	games, err := h.DB.GetOngoingGames()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "server error"})
		return
	}

	if games == nil {
		games = []models.Game{}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"games": games,
	})
}
