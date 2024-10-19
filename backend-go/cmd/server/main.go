package main

import (
        "encoding/json"
        "fourinrow/internal/database"
        "fourinrow/internal/game"
        "fourinrow/internal/kafka"
        "fourinrow/internal/matchmaking"
        "fourinrow/internal/websocket"
        "log"
        "net/http"
        "os"
        "os/signal"
        "path/filepath"
        "syscall"
        "time"

        "github.com/gorilla/mux"
        ws "github.com/gorilla/websocket"
)

var upgrader = ws.Upgrader{
        CheckOrigin: func(r *http.Request) bool {
                return true
        },
}

func main() {
        log.Println("🚀 Starting 4 in a Row server (Go backend)...")

        port := os.Getenv("PORT")
        if port == "" {
                port = "8080"
        }

        db, err := database.NewDB()
        if err != nil {
                log.Fatalf("Failed to connect to database: %v", err)
        }
        defer db.Close()

        if err := db.Initialize(); err != nil {
                log.Fatalf("Failed to initialize database: %v", err)
        }

        kafkaProducer, err := kafka.NewProducer()
        if err != nil {
                log.Fatalf("Failed to create Kafka producer: %v", err)
        }
        defer kafkaProducer.Close()

        matchmaker := matchmaking.NewMatchmaker(10*time.Second, 30*time.Second)
        hub := websocket.NewHub(matchmaker)

        hub.SetGameEventCallback(func(eventType string, data interface{}) {
                if err := kafkaProducer.ProduceEvent(eventType, data); err != nil {
                        log.Printf("Failed to produce Kafka event: %v", err)
                }
                
                if eventType == "game_ended" {
                        if gameState, ok := data.(*game.GameState); ok {
                                if err := db.SaveGame(gameState); err != nil {
                                        log.Printf("Failed to save game: %v", err)
                                }
                        }
                }
        })

        go hub.Run()

        router := mux.NewRouter()

        router.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
                conn, err := upgrader.Upgrade(w, r, nil)
                if err != nil {
                        log.Printf("WebSocket upgrade error: %v", err)
                        return
                }
                websocket.ServeWS(hub, conn)
        })

        router.HandleFunc("/api/health", func(w http.ResponseWriter, r *http.Request) {
                w.Header().Set("Content-Type", "application/json")
                if err := json.NewEncoder(w).Encode(map[string]string{"status": "ok"}); err != nil {
                        log.Printf("Failed to encode health response: %v", err)
                }
        }).Methods("GET")

        router.HandleFunc("/api/leaderboard", func(w http.ResponseWriter, r *http.Request) {
                stats, err := db.GetLeaderboard(10)
                if err != nil {
                        http.Error(w, err.Error(), http.StatusInternalServerError)
                        return
                }
                w.Header().Set("Content-Type", "application/json")
                if err := json.NewEncoder(w).Encode(stats); err != nil {
                        log.Printf("Failed to encode leaderboard response: %v", err)
                }
        }).Methods("GET")

        frontendPath := filepath.Join("..", "frontend", "dist")
        fs := http.FileServer(http.Dir(frontendPath))
        router.PathPrefix("/").Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
                path := filepath.Join(frontendPath, r.URL.Path)
                if _, err := os.Stat(path); os.IsNotExist(err) {
                        http.ServeFile(w, r, filepath.Join(frontendPath, "index.html"))
                        return
                }
                fs.ServeHTTP(w, r)
        }))

        log.Printf("✅ Server running on http://0.0.0.0:%s", port)
        log.Printf("   WebSocket: ws://0.0.0.0:%s/ws", port)

        srv := &http.Server{
                Addr:    "0.0.0.0:" + port,
                Handler: router,
        }

        go func() {
                if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
                        log.Fatalf("Server error: %v", err)
                }
        }()

        quit := make(chan os.Signal, 1)
        signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
        <-quit

        log.Println("Shutting down gracefully...")
}
