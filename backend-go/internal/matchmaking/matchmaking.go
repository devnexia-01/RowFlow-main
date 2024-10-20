package matchmaking

import (
        "fourinrow/internal/bot"
        "fourinrow/internal/game"
        "log"
        "sync"
        "time"

        "github.com/google/uuid"
)

type ClientConnection struct {
        ID           string
        Username     string
        GameID       string
        PlayerNumber game.Player
}

type Matchmaker struct {
        mu                   sync.RWMutex
        waitingPlayers       []*ClientConnection
        games                map[string]*game.GameState
        playerToGame         map[string]string
        reconnectionTimeout  time.Duration
        matchmakingTimeout   time.Duration
        onGameCreated        func(*game.GameState)
}

func NewMatchmaker(matchmakingTimeout, reconnectionTimeout time.Duration) *Matchmaker {
        return &Matchmaker{
                waitingPlayers:      make([]*ClientConnection, 0),
                games:               make(map[string]*game.GameState),
                playerToGame:        make(map[string]string),
                reconnectionTimeout: reconnectionTimeout,
                matchmakingTimeout:  matchmakingTimeout,
        }
}

func (m *Matchmaker) SetGameCreatedCallback(callback func(*game.GameState)) {
        m.onGameCreated = callback
}

func (m *Matchmaker) AddToQueue(client *ClientConnection) {
        m.mu.Lock()
        m.waitingPlayers = append(m.waitingPlayers, client)
        log.Printf("Player %s added to matchmaking queue", client.Username)
        m.mu.Unlock()

        m.tryMatch(client)
}

func (m *Matchmaker) tryMatch(client *ClientConnection) {
        m.mu.Lock()
        defer m.mu.Unlock()

        var otherPlayer *ClientConnection
        for _, p := range m.waitingPlayers {
                if p.ID != client.ID && p.GameID == "" {
                        otherPlayer = p
                        break
                }
        }

        if otherPlayer != nil {
                gameState := m.createGame(client, otherPlayer)
                if m.onGameCreated != nil {
                        go m.onGameCreated(gameState)
                }
        } else {
                go func() {
                        time.Sleep(m.matchmakingTimeout)
                        m.mu.Lock()
                        defer m.mu.Unlock()

                        if client.GameID == "" {
                                for _, p := range m.waitingPlayers {
                                        if p.ID == client.ID {
                                                log.Printf("Matching %s with bot after timeout", client.Username)
                                                gameState := m.createGameWithBot(client)
                                                if m.onGameCreated != nil {
                                                        go m.onGameCreated(gameState)
                                                }
                                                break
                                        }
                                }
                        }
                }()
        }
}

func (m *Matchmaker) createGame(player1, player2 *ClientConnection) *game.GameState {
        gameID := uuid.New().String()

        gameState := &game.GameState{
                ID:          gameID,
                Player1:     player1.Username,
                Player2:     player2.Username,
                Board:       game.CreateBoard(),
                CurrentTurn: game.Player1,
                IsFinished:  false,
        }

        m.games[gameID] = gameState

        player1.GameID = gameID
        player1.PlayerNumber = game.Player1
        player2.GameID = gameID
        player2.PlayerNumber = game.Player2

        m.playerToGame[player1.Username] = gameID
        m.playerToGame[player2.Username] = gameID

        newWaiting := make([]*ClientConnection, 0)
        for _, p := range m.waitingPlayers {
                if p.ID != player1.ID && p.ID != player2.ID {
                        newWaiting = append(newWaiting, p)
                }
        }
        m.waitingPlayers = newWaiting

        log.Printf("Game %s created: %s vs %s", gameID, player1.Username, player2.Username)
        return gameState
}

func (m *Matchmaker) createGameWithBot(player *ClientConnection) *game.GameState {
        gameID := uuid.New().String()

        gameState := &game.GameState{
                ID:          gameID,
                Player1:     player.Username,
                Player2:     bot.BotUsername,
                Board:       game.CreateBoard(),
                CurrentTurn: game.Player1,
                IsFinished:  false,
        }

        m.games[gameID] = gameState

        player.GameID = gameID
        player.PlayerNumber = game.Player1

        m.playerToGame[player.Username] = gameID

        newWaiting := make([]*ClientConnection, 0)
        for _, p := range m.waitingPlayers {
                if p.ID != player.ID {
                        newWaiting = append(newWaiting, p)
                }
        }
        m.waitingPlayers = newWaiting

        log.Printf("Game %s created: %s vs Bot", gameID, player.Username)
        return gameState
}

func (m *Matchmaker) GetGame(gameID string) (*game.GameState, bool) {
        m.mu.RLock()
        defer m.mu.RUnlock()
        gameState, exists := m.games[gameID]
        return gameState, exists
}

func (m *Matchmaker) GetGameByPlayer(username string) (*game.GameState, bool) {
        m.mu.RLock()
        defer m.mu.RUnlock()
        gameID, exists := m.playerToGame[username]
        if !exists {
                return nil, false
        }
        gameState, exists := m.games[gameID]
        return gameState, exists
}

func (m *Matchmaker) UpdateGame(gameID string, gameState *game.GameState) {
        m.mu.Lock()
        defer m.mu.Unlock()
        m.games[gameID] = gameState
}

func (m *Matchmaker) RemoveGame(gameID string) {
        m.mu.Lock()
        defer m.mu.Unlock()

        if gameState, exists := m.games[gameID]; exists {
                delete(m.playerToGame, gameState.Player1)
                delete(m.playerToGame, gameState.Player2)
                delete(m.games, gameID)
                log.Printf("Game %s removed", gameID)
        }
}

func (m *Matchmaker) RemoveFromQueue(clientID string) {
        m.mu.Lock()
        defer m.mu.Unlock()

        newWaiting := make([]*ClientConnection, 0)
        for _, p := range m.waitingPlayers {
                if p.ID != clientID {
                        newWaiting = append(newWaiting, p)
                }
        }
        m.waitingPlayers = newWaiting
}

func (m *Matchmaker) GetAllGames() []*game.GameState {
        m.mu.RLock()
        defer m.mu.RUnlock()

        games := make([]*game.GameState, 0, len(m.games))
        for _, gameState := range m.games {
                games = append(games, gameState)
        }
        return games
}
