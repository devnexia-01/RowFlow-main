function Leaderboard({ leaderboard }) {
  if (!leaderboard || leaderboard.length === 0) {
    return null
  }

  return (
    <div className="leaderboard">
      <h2>🏆 Leaderboard</h2>
      <ul className="leaderboard-list">
        {leaderboard.map((player, index) => {
          const totalGames = (player.wins || 0) + (player.losses || 0) + (player.draws || 0)
          return (
            <li key={player.username} className="leaderboard-item">
              <div>
                <span className="rank">#{index + 1}</span>
                <strong>{player.username}</strong>
              </div>
              <div className="stats">
                <span>Wins: {player.wins || 0}</span>
                <span>Games: {totalGames}</span>
              </div>
            </li>
          )
        })}
      </ul>
    </div>
  )
}

export default Leaderboard
