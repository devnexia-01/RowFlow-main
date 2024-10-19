package websocket

import (
        "encoding/json"
        "fourinrow/internal/bot"
        "fourinrow/internal/game"
        "fourinrow/internal/matchmaking"
        "log"
        "sync"
        "time"

        "github.com/google/uuid"
        "github.com/gorilla/websocket"
)

type Client struct {
        ID             string
        Hub            *Hub
        Conn           *websocket.Conn
        Send           chan []byte
        Username       string
        GameID         string
        PlayerNumber   game.Player
        Disconnected   bool
        DisconnectedAt time.Time
}

type Hub struct {
        clients      map[*Client]bool
        broadcast    chan []byte
        register     chan *Client
        unregister   chan *Client
        mu           sync.RWMutex
        matchmaker   *matchmaking.Matchmaker
        onGameEvent  func(string, interface{})
}

type Message struct {
        Type     string      `json:"type"`
        Data     interface{} `json:"data,omitempty"`
        Username string      `json:"username,omitempty"`
        Column   int         `json:"column,omitempty"`
        GameID   string      `json:"gameId,omitempty"`
}

func NewHub(matchmaker *matchmaking.Matchmaker) *Hub {
        hub := &Hub{
                broadcast:  make(chan []byte, 256),
                register:   make(chan *Client),
                unregister: make(chan *Client),
                clients:    make(map[*Client]bool),
                matchmaker: matchmaker,
        }

        matchmaker.SetGameCreatedCallback(func(gameState *game.GameState) {
                hub.handleGameCreated(gameState)
        })

        return hub
}

func (h *Hub) SetGameEventCallback(callback func(string, interface{})) {
        h.onGameEvent = callback
}

func (h *Hub) Run() {
        for {
                select {
                case client := <-h.register:
                        h.mu.Lock()
                        h.clients[client] = true
                        h.mu.Unlock()
                        log.Printf("Client registered: %s", client.ID)

                case client := <-h.unregister:
                        h.mu.Lock()
                        if _, ok := h.clients[client]; ok {
                                if client.GameID != "" {
                                        client.Disconnected = true
                                        client.DisconnectedAt = time.Now()
                                        log.Printf("Client disconnected: %s (username: %s), will wait 30s for reconnection", client.ID, client.Username)
                                        go h.handleDisconnectionTimeout(client)
                                } else {
                                        delete(h.clients, client)
                                        close(client.Send)
                                        h.matchmaker.RemoveFromQueue(client.ID)
                                        log.Printf("Client unregistered: %s", client.ID)
                                }
                        }
                        h.mu.Unlock()

                case message := <-h.broadcast:
                        h.mu.RLock()
                        deadClients := []*Client{}
                        for client := range h.clients {
                                select {
                                case client.Send <- message:
                                default:
                                        deadClients = append(deadClients, client)
                                }
                        }
                        h.mu.RUnlock()
                        
                        if len(deadClients) > 0 {
                                h.mu.Lock()
                                for _, client := range deadClients {
                                        if _, ok := h.clients[client]; ok {
                                                close(client.Send)
                                                delete(h.clients, client)
                                        }
                                }
                                h.mu.Unlock()
                        }
                }
        }
}

func (h *Hub) handleGameCreated(gameState *game.GameState) {
        h.mu.RLock()
        defer h.mu.RUnlock()

        for client := range h.clients {
                if client.Username == gameState.Player1 || client.Username == gameState.Player2 {
                        yourTurn := client.Username == gameState.Player1
                        response := Message{
                                Type: "game_start",
                                Data: map[string]interface{}{
                                        "gameId":  gameState.ID,
                                        "player1": gameState.Player1,
                                        "player2": gameState.Player2,
                                        "yourTurn": yourTurn,
                                },
                        }
                        responseBytes, _ := json.Marshal(response)
                        select {
                        case client.Send <- responseBytes:
                        default:
                        }
                }
        }

        if h.onGameEvent != nil {
                h.onGameEvent("game_started", gameState)
        }
}

func (h *Hub) HandleJoin(client *Client, username string) {
        client.Username = username

        conn := &matchmaking.ClientConnection{
                ID:       client.ID,
                Username: username,
        }

        h.matchmaker.AddToQueue(conn)

        response := Message{
                Type: "waiting",
                Data: map[string]string{"message": "Waiting for opponent..."},
        }
        responseBytes, _ := json.Marshal(response)
        client.Send <- responseBytes
}

func (h *Hub) HandleMove(client *Client, column int) {
        gameState, exists := h.matchmaker.GetGameByPlayer(client.Username)
        if !exists {
                h.sendError(client, "No active game found")
                return
        }

        if gameState.IsFinished {
                h.sendError(client, "Game is already finished")
                return
        }

        playerNumber := game.Player1
        if gameState.Player1 == client.Username {
                playerNumber = game.Player1
        } else if gameState.Player2 == client.Username {
                playerNumber = game.Player2
        }

        if gameState.CurrentTurn != playerNumber {
                h.sendError(client, "Not your turn")
                return
        }

        move, err := game.MakeMove(&gameState.Board, column, playerNumber)
        if err != nil {
                h.sendError(client, err.Error())
                return
        }

        gameState.CurrentTurn = game.Player1
        if playerNumber == game.Player1 {
                gameState.CurrentTurn = game.Player2
        }

        h.matchmaker.UpdateGame(gameState.ID, gameState)

        h.broadcastMove(gameState, move)

        if h.onGameEvent != nil {
                h.onGameEvent("move_made", map[string]interface{}{
                        "gameId": gameState.ID,
                        "player": client.Username,
                        "move":   move,
                })
        }

        winner, isDraw := game.CheckWinner(&gameState.Board)
        if winner != game.Empty || isDraw {
                h.handleGameEnd(gameState, winner, isDraw)
                return
        }

        // Bot's turn
        if gameState.Player2 == bot.BotUsername && gameState.CurrentTurn == game.Player2 {
                h.handleBotMove(gameState)
        }
}

func (h *Hub) handleBotMove(gameState *game.GameState) {
        botColumn := bot.SelectBotMove(&gameState.Board, game.Player2)
        move, _ := game.MakeMove(&gameState.Board, botColumn, game.Player2)

        gameState.CurrentTurn = game.Player1
        h.matchmaker.UpdateGame(gameState.ID, gameState)

        h.broadcastMove(gameState, move)

        if h.onGameEvent != nil {
                h.onGameEvent("move_made", map[string]interface{}{
                        "gameId": gameState.ID,
                        "player": bot.BotUsername,
                        "move":   move,
                })
        }

        winner, isDraw := game.CheckWinner(&gameState.Board)
        if winner != game.Empty || isDraw {
                h.handleGameEnd(gameState, winner, isDraw)
        }
}

func (h *Hub) handleGameEnd(gameState *game.GameState, winner game.Player, isDraw bool) {
        gameState.IsFinished = true

        if isDraw {
                gameState.Winner = "Draw"
        } else if winner == game.Player1 {
                gameState.Winner = gameState.Player1
        } else {
                gameState.Winner = gameState.Player2
        }

        h.matchmaker.UpdateGame(gameState.ID, gameState)

        response := Message{
                Type: "game_over",
                Data: map[string]interface{}{
                        "winner": gameState.Winner,
                },
        }
        responseBytes, _ := json.Marshal(response)

        h.mu.RLock()
        for client := range h.clients {
                if client.Username == gameState.Player1 || client.Username == gameState.Player2 {
                        select {
                        case client.Send <- responseBytes:
                        default:
                        }
                }
        }
        h.mu.RUnlock()

        if h.onGameEvent != nil {
                h.onGameEvent("game_ended", gameState)
        }
}

func (h *Hub) broadcastMove(gameState *game.GameState, move *game.Move) {
        response := Message{
                Type: "move",
                Data: map[string]interface{}{
                        "row":    move.Row,
                        "column": move.Column,
                        "player": move.Player,
                },
        }
        responseBytes, _ := json.Marshal(response)

        h.mu.RLock()
        for client := range h.clients {
                if client.Username == gameState.Player1 || client.Username == gameState.Player2 {
                        select {
                        case client.Send <- responseBytes:
                        default:
                        }
                }
        }
        h.mu.RUnlock()
}

func (h *Hub) sendError(client *Client, message string) {
        response := map[string]string{
                "type":  "error",
                "error": message,
        }
        responseBytes, _ := json.Marshal(response)
        client.Send <- responseBytes
}

func (h *Hub) handleDisconnectionTimeout(client *Client) {
        time.Sleep(30 * time.Second)

        h.mu.Lock()
        defer h.mu.Unlock()

        if _, stillExists := h.clients[client]; !stillExists {
                return
        }

        if client.Disconnected {
                log.Printf("Player %s did not reconnect within 30 seconds, forfeiting game", client.Username)
                
                gameState, exists := h.matchmaker.GetGameByPlayer(client.Username)
                if exists && !gameState.IsFinished {
                        var opponent string
                        if gameState.Player1 == client.Username {
                                opponent = gameState.Player2
                        } else {
                                opponent = gameState.Player1
                        }

                        gameState.IsFinished = true
                        gameState.Winner = opponent
                        h.matchmaker.UpdateGame(gameState.ID, gameState)

                        response := Message{
                                Type: "game_over",
                                Data: map[string]interface{}{
                                        "winner": opponent,
                                        "reason": "opponent_disconnected",
                                },
                        }
                        responseBytes, _ := json.Marshal(response)

                        for c := range h.clients {
                                if c.Username == opponent && !c.Disconnected {
                                        select {
                                        case c.Send <- responseBytes:
                                        default:
                                        }
                                }
                        }

                        if h.onGameEvent != nil {
                                h.onGameEvent("game_ended", gameState)
                        }
                }

                delete(h.clients, client)
                if client.Send != nil {
                        close(client.Send)
                }
        }
}

func (c *Client) ReadPump() {
        defer func() {
                c.Hub.unregister <- c
                c.Conn.Close()
        }()

        for {
                _, message, err := c.Conn.ReadMessage()
                if err != nil {
                        break
                }

                var msg Message
                if err := json.Unmarshal(message, &msg); err != nil {
                        log.Printf("Error unmarshaling message: %v", err)
                        continue
                }

                switch msg.Type {
                case "join":
                        c.Hub.HandleJoin(c, msg.Username)
                case "move":
                        c.Hub.HandleMove(c, msg.Column)
                }
        }
}

func (c *Client) WritePump() {
        defer c.Conn.Close()

        for message := range c.Send {
                err := c.Conn.WriteMessage(websocket.TextMessage, message)
                if err != nil {
                        break
                }
        }
}

func ServeWS(hub *Hub, conn *websocket.Conn) {
        client := &Client{
                ID:   uuid.New().String(),
                Hub:  hub,
                Conn: conn,
                Send: make(chan []byte, 256),
        }

        hub.register <- client

        go client.WritePump()
        go client.ReadPump()
}
