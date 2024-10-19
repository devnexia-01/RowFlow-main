function GameBoard({ board, onMove, disabled }) {
  const handleColumnClick = (col) => {
    if (!disabled) {
      onMove(col)
    }
  }

  return (
    <div className="board">
      {board.map((row, rowIndex) => (
        <div key={rowIndex} className="row">
          {row.map((cell, colIndex) => (
            <div
              key={`${rowIndex}-${colIndex}`}
              className={`cell ${
                cell === 1 ? 'player1' : cell === 2 ? 'player2' : ''
              } ${disabled ? 'disabled' : ''}`}
              onClick={() => !disabled && rowIndex === 0 && handleColumnClick(colIndex)}
            />
          ))}
        </div>
      ))}
    </div>
  )
}

export default GameBoard
