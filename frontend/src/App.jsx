import { useState, useEffect } from 'react'
import GameBoard from './components/GameBoard'
import Leaderboard from './components/Leaderboard'
import useWebSocket from './hooks/useWebSocket'

function App() {
  const [username, setUsername] = useState('')
  const [hasJoined, setHasJoined] = useState(false)
  const [gameState, setGameState] = useState(null)
  const [error, setError] = useState('')
  const [leaderboard, setLeaderboard] = useState([])

  const { sendMessage, lastMessage, connectionStatus } = useWebSocket()

  useEffect(() => {
    if (lastMessage) {
      handleMessage(lastMessage)
    }
  }, [lastMessage])

  useEffect(() => {
    fetchLeaderboard()
    const interval = setInterval(fetchLeaderboard, 10000)
    return () => clearInterval(interval)
  }, [])

  const fetchLeaderboard = async () => {
    try {
      const response = await fetch('/api/leaderboard')
      const data = await response.json()
      setLeaderboard(data || [])
    } catch (err) {
      setLeaderboard([])
    }
  }

  const handleMessage = (message) => {
    const msg = typeof message === 'string' ? JSON.parse(message) : message

    switch (msg.type) {
      case 'waiting':
        setGameState({ status: 'waiting' })
        break

      case 'game_start':
        setGameState({
          status: 'playing',
          gameId: msg.data.gameId,
          player1: msg.data.player1,
          player2: msg.data.player2,
          board: Array(6).fill(null).map(() => Array(7).fill(0)),
          currentTurn: 1,
          yourTurn: msg.data.yourTurn,
          playerNumber: msg.data.yourTurn ? 1 : 2
        })
        break

      case 'move':
        if (gameState && gameState.board) {
          const newBoard = gameState.board.map(row => [...row])
          newBoard[msg.data.row][msg.data.column] = msg.data.player
          
          setGameState({
            ...gameState,
            board: newBoard,
            currentTurn: msg.data.player === 1 ? 2 : 1,
            yourTurn: gameState.playerNumber !== msg.data.player
          })
        }
        break

      case 'game_over':
        if (gameState) {
          setGameState({
            ...gameState,
            status: 'finished',
            winner: msg.data.winner,
            reason: msg.data.reason
          })
          fetchLeaderboard()
        }
        break

      case 'error':
        setError(msg.error)
        setTimeout(() => setError(''), 5000)
        break

      case 'reconnected':
        if (msg.data.board) {
          setGameState({
            ...gameState,
            board: msg.data.board,
            currentTurn: msg.data.turn,
            status: 'playing'
          })
        }
        break
    }
  }

  const handleJoin = (e) => {
    e.preventDefault()
    if (username.trim()) {
      sendMessage({
        type: 'join',
        username: username.trim()
      })
      setHasJoined(true)
    }
  }

  const handleMove = (column) => {
    if (gameState && gameState.yourTurn && gameState.status === 'playing') {
      sendMessage({
        type: 'move',
        column: column
      })
    }
  }

  const handleNewGame = () => {
    setHasJoined(false)
    setGameState(null)
    setUsername('')
    setError('')
  }

  return (
    <div className="app">
      <div className="game-container">
        <div className="game-header">
          <h1>🎮 4 in a Row</h1>
          <p className="game-status">
            {connectionStatus === 'connected' ? '🟢 Connected' : '🔴 Disconnected'}
          </p>
        </div>

        {error && <div className="error-message">{error}</div>}

        {!hasJoined ? (
          <form className="join-form" onSubmit={handleJoin}>
            <input
              type="text"
              placeholder="Enter your username"
              value={username}
              onChange={(e) => setUsername(e.target.value)}
              maxLength={20}
              required
            />
            <button type="submit">Join Game</button>
          </form>
        ) : gameState ? (
          <>
            {gameState.status === 'waiting' && (
              <div className="waiting-message">
                Waiting for opponent... (Bot will join in 10 seconds)
              </div>
            )}

            {gameState.status === 'playing' && (
              <>
                <div className="player-info">
                  <div className={`player ${gameState.currentTurn === 1 ? 'active' : ''}`}>
                    <strong>🔴 {gameState.player1}</strong>
                    {gameState.playerNumber === 1 && ' (You)'}
                  </div>
                  <div className={`player ${gameState.currentTurn === 2 ? 'active' : ''}`}>
                    <strong>🟡 {gameState.player2}</strong>
                    {gameState.playerNumber === 2 && ' (You)'}
                  </div>
                </div>

                <GameBoard
                  board={gameState.board}
                  onMove={handleMove}
                  disabled={!gameState.yourTurn}
                />

                <div className="game-info">
                  {gameState.yourTurn ? (
                    <p><strong>Your turn!</strong> Click a column to drop your disc.</p>
                  ) : (
                    <p>Waiting for opponent's move...</p>
                  )}
                </div>
              </>
            )}

            {gameState.status === 'finished' && (
              <>
                <GameBoard
                  board={gameState.board}
                  onMove={() => {}}
                  disabled={true}
                />
                
                <div className="winner-message">
                  {gameState.winner === 'Draw'
                    ? "🤝 It's a draw!"
                    : gameState.winner === username
                    ? '🎉 You won!'
                    : `${gameState.winner} won!`}
                </div>

                {gameState.reason === 'opponent_disconnected' && (
                  <p>Opponent disconnected</p>
                )}

                <button className="new-game-btn" onClick={handleNewGame}>
                  Play Again
                </button>
              </>
            )}
          </>
        ) : (
          <div className="waiting-message">Joining game...</div>
        )}
      </div>

      <Leaderboard leaderboard={leaderboard} />
    </div>
  )
}

export default App
