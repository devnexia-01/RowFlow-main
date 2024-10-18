# 4 in a Row

Connect Four game with multiplayer and bot support. Built with Go backend and React frontend.

## Features

- Play against other players online
- AI bot opponent
- Real-time gameplay using WebSockets
- Leaderboard with player stats
- PostgreSQL database for game history

## Tech Stack

**Backend:**
- Go 1.24
- gorilla/websocket for WebSocket support
- gorilla/mux for routing
- PostgreSQL database
- Kafka for event streaming (optional)

**Frontend:**
- React 18
- Vite
- WebSocket API

## Running the Project

### Prerequisites
- Go 1.24+
- Node.js 20+
- PostgreSQL (optional - works without it)

### Setup

1. Install dependencies:
```bash
cd backend-go
go mod download

cd ../frontend
npm install
```

2. Build frontend:
```bash
cd frontend
npm run build
```

3. Run the server:
```bash
cd backend-go
PORT=5000 go run ./cmd/server
```

The game will be available at http://localhost:5000

### Environment Variables

- `PORT` - Server port (default: 8080)
- `DATABASE_URL` - PostgreSQL connection string (optional)
- `KAFKA_ENABLED` - Enable Kafka events (default: false)
- `KAFKA_BROKER` - Kafka broker address

## How It Works

### Matchmaking
- Players join and wait for an opponent
- If no player available after 10 seconds, bot joins
- Games start automatically when 2 players are matched

### Game Rules
- 7 columns × 6 rows board
- Connect 4 discs horizontally, vertically, or diagonally to win
- Players alternate turns
- Column must have empty space to place disc

### Bot AI
The bot uses a simple strategy:
1. Try to win if possible
2. Block opponent's winning move
3. Make strategic moves (create opportunities)
4. Prefer center columns
5. Random valid move as fallback

## API Endpoints

- `GET /api/health` - Health check
- `GET /api/leaderboard` - Top 10 players
- `WS /ws` - WebSocket connection for gameplay


```
backend-go/
├── cmd/server/          # Main server entry
├── internal/
│   ├── game/           # Game logic
│   ├── bot/            # AI bot
│   ├── matchmaking/    # Player matching
│   ├── websocket/      # WebSocket handler
│   ├── database/       # Database layer
│   └── kafka/          # Kafka producer
└── go.mod

frontend/
├── src/
│   ├── components/     # React components
│   ├── hooks/         # Custom hooks
│   └── App.jsx        # Main app
└── package.json
```
