package database

import (
        "database/sql"
        "fourinrow/internal/game"
        "log"
        "os"

        _ "github.com/lib/pq"
)

type DB struct {
        conn *sql.DB
}

type PlayerStats struct {
        Username  string `json:"username"`
        Wins      int    `json:"wins"`
        Losses    int    `json:"losses"`
        Draws     int    `json:"draws"`
}

func NewDB() (*DB, error) {
        dbURL := os.Getenv("DATABASE_URL")
        if dbURL == "" {
                log.Println("DATABASE_URL not set, database features disabled")
                return &DB{conn: nil}, nil
        }

        conn, err := sql.Open("postgres", dbURL)
        if err != nil {
                return nil, err
        }

        if err := conn.Ping(); err != nil {
                return nil, err
        }

        log.Println("✅ Database connection established")
        return &DB{conn: conn}, nil
}

func (db *DB) Initialize() error {
        if db.conn == nil {
                return nil
        }

        createPlayersTable := `
        CREATE TABLE IF NOT EXISTS players (
                username VARCHAR(255) PRIMARY KEY,
                wins INTEGER DEFAULT 0,
                losses INTEGER DEFAULT 0,
                draws INTEGER DEFAULT 0,
                created_at TIMESTAMP DEFAULT NOW()
        );`

        createGamesTable := `
        CREATE TABLE IF NOT EXISTS games (
                id SERIAL PRIMARY KEY,
                game_id VARCHAR(255) UNIQUE,
                player1 VARCHAR(255),
                player2 VARCHAR(255),
                winner VARCHAR(255),
                moves_data TEXT,
                created_at TIMESTAMP DEFAULT NOW()
        );`

        if _, err := db.conn.Exec(createPlayersTable); err != nil {
                return err
        }

        if _, err := db.conn.Exec(createGamesTable); err != nil {
                return err
        }

        log.Println("✅ Database tables initialized")
        return nil
}

func (db *DB) SaveGame(gameState *game.GameState) error {
        if db.conn == nil {
                return nil
        }

        _, err := db.conn.Exec(
                `INSERT INTO games (game_id, player1, player2, winner) 
                 VALUES ($1, $2, $3, $4)`,
                gameState.ID, gameState.Player1, gameState.Player2, gameState.Winner,
        )

        if err != nil {
                return err
        }

        if gameState.Winner == "Draw" {
                if err := db.updatePlayerStats(gameState.Player1, 0, 0, 1); err != nil {
                        log.Printf("Failed to update player stats for %s: %v", gameState.Player1, err)
                }
                if gameState.Player2 != "AI Bot" {
                        if err := db.updatePlayerStats(gameState.Player2, 0, 0, 1); err != nil {
                                log.Printf("Failed to update player stats for %s: %v", gameState.Player2, err)
                        }
                }
        } else if gameState.Winner == gameState.Player1 {
                if err := db.updatePlayerStats(gameState.Player1, 1, 0, 0); err != nil {
                        log.Printf("Failed to update player stats for %s: %v", gameState.Player1, err)
                }
                if gameState.Player2 != "AI Bot" {
                        if err := db.updatePlayerStats(gameState.Player2, 0, 1, 0); err != nil {
                                log.Printf("Failed to update player stats for %s: %v", gameState.Player2, err)
                        }
                }
        } else if gameState.Winner == gameState.Player2 {
                if gameState.Player2 != "AI Bot" {
                        if err := db.updatePlayerStats(gameState.Player2, 1, 0, 0); err != nil {
                                log.Printf("Failed to update player stats for %s: %v", gameState.Player2, err)
                        }
                }
                if err := db.updatePlayerStats(gameState.Player1, 0, 1, 0); err != nil {
                        log.Printf("Failed to update player stats for %s: %v", gameState.Player1, err)
                }
        }

        return nil
}

func (db *DB) updatePlayerStats(username string, wins, losses, draws int) error {
        if db.conn == nil {
                return nil
        }

        _, err := db.conn.Exec(
                `INSERT INTO players (username, wins, losses, draws)
                 VALUES ($1, $2, $3, $4)
                 ON CONFLICT (username) 
                 DO UPDATE SET 
                   wins = players.wins + $2,
                   losses = players.losses + $3,
                   draws = players.draws + $4`,
                username, wins, losses, draws,
        )

        return err
}

func (db *DB) GetLeaderboard(limit int) ([]PlayerStats, error) {
        if db.conn == nil {
                return []PlayerStats{}, nil
        }

        rows, err := db.conn.Query(
                `SELECT username, wins, losses, draws 
                 FROM players 
                 ORDER BY wins DESC 
                 LIMIT $1`,
                limit,
        )
        if err != nil {
                return nil, err
        }
        defer rows.Close()

        stats := []PlayerStats{}
        for rows.Next() {
                var s PlayerStats
                if err := rows.Scan(&s.Username, &s.Wins, &s.Losses, &s.Draws); err != nil {
                        return nil, err
                }
                stats = append(stats, s)
        }

        return stats, nil
}

func (db *DB) Close() error {
        if db.conn == nil {
                return nil
        }
        return db.conn.Close()
}
