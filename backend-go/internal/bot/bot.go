package bot

import (
        "fourinrow/internal/game"
        "math"
        "math/rand"
)

const BotUsername = "AI Bot"

func SelectBotMove(board *game.Board, botPlayer game.Player) int {
        opponent := game.Player1
        if botPlayer == game.Player1 {
                opponent = game.Player2
        }

        if winningMove := findWinningMove(board, botPlayer); winningMove != -1 {
                return winningMove
        }

        if blockingMove := findWinningMove(board, opponent); blockingMove != -1 {
                return blockingMove
        }

        if strategicMove := findStrategicMove(board, botPlayer); strategicMove != -1 {
                return strategicMove
        }

        validCols := game.GetValidColumns(board)
        centerCols := []int{}
        for _, col := range validCols {
                if col >= 2 && col <= 4 {
                        centerCols = append(centerCols, col)
                }
        }
        if len(centerCols) > 0 {
                return centerCols[rand.Intn(len(centerCols))]
        }

        return validCols[rand.Intn(len(validCols))]
}

func findWinningMove(board *game.Board, player game.Player) int {
        validCols := game.GetValidColumns(board)

        for _, col := range validCols {
                testBoard := *board
                _, err := game.MakeMove(&testBoard, col, player)
                if err != nil {
                        continue
                }
                winner, _ := game.CheckWinner(&testBoard)
                if winner == player {
                        return col
                }
        }

        return -1
}

func findStrategicMove(board *game.Board, player game.Player) int {
        validCols := game.GetValidColumns(board)
        bestScore := math.Inf(-1)
        bestCol := -1

        for _, col := range validCols {
                score := evaluateColumn(board, col, player)
                if score > bestScore {
                        bestScore = score
                        bestCol = col
                }
        }

        return bestCol
}

func evaluateColumn(board *game.Board, column int, player game.Player) float64 {
        testBoard := *board
        move, err := game.MakeMove(&testBoard, column, player)
        if err != nil {
                return math.Inf(-1)
        }

        row := move.Row
        score := 0.0

        score += evaluateLine(&testBoard, row, column, 0, 1, player)
        score += evaluateLine(&testBoard, row, column, 1, 0, player)
        score += evaluateLine(&testBoard, row, column, 1, 1, player)
        score += evaluateLine(&testBoard, row, column, 1, -1, player)

        return score
}

func evaluateLine(board *game.Board, row, col, dRow, dCol int, player game.Player) float64 {
        count := 0
        empty := 0

        for dir := -1; dir <= 1; dir += 2 {
                for i := 1; i < 4; i++ {
                        r := row + dRow*i*dir
                        c := col + dCol*i*dir

                        if r < 0 || r >= game.Rows || c < 0 || c >= game.Cols {
                                break
                        }

                        cell := board[r][c]
                        if cell == player {
                                count++
                        } else if cell == game.Empty {
                                empty++
                                break
                        } else {
                                break
                        }
                }
        }

        if count >= 2 && empty >= 1 {
                return float64(count * 10)
        }
        if count >= 1 && empty >= 2 {
                return float64(count * 5)
        }
        return float64(count)
}
