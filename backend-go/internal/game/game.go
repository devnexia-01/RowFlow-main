package game

import (
        "errors"
)

const (
        Rows      = 6
        Cols      = 7
        WinLength = 4
)

type Player int

const (
        Empty  Player = 0
        Player1 Player = 1
        Player2 Player = 2
)

type Board [Rows][Cols]Player

type Move struct {
        Column int    `json:"column"`
        Row    int    `json:"row"`
        Player Player `json:"player"`
}

type GameState struct {
        ID         string `json:"id"`
        Player1    string `json:"player1"`
        Player2    string `json:"player2"`
        Board      Board  `json:"board"`
        CurrentTurn Player `json:"currentTurn"`
        Winner     string `json:"winner,omitempty"`
        IsFinished bool   `json:"isFinished"`
}

func CreateBoard() Board {
        return Board{}
}

func MakeMove(board *Board, column int, player Player) (*Move, error) {
        if column < 0 || column >= Cols {
                return nil, errors.New("invalid column")
        }

        for row := Rows - 1; row >= 0; row-- {
                if board[row][column] == Empty {
                        board[row][column] = player
                        return &Move{
                                Column: column,
                                Row:    row,
                                Player: player,
                        }, nil
                }
        }

        return nil, errors.New("column is full")
}

func CheckWinner(board *Board) (Player, bool) {
        for row := 0; row < Rows; row++ {
                for col := 0; col <= Cols-WinLength; col++ {
                        player := board[row][col]
                        if player != Empty {
                                win := true
                                for i := 1; i < WinLength; i++ {
                                        if board[row][col+i] != player {
                                                win = false
                                                break
                                        }
                                }
                                if win {
                                        return player, false
                                }
                        }
                }
        }

        for col := 0; col < Cols; col++ {
                for row := 0; row <= Rows-WinLength; row++ {
                        player := board[row][col]
                        if player != Empty {
                                win := true
                                for i := 1; i < WinLength; i++ {
                                        if board[row+i][col] != player {
                                                win = false
                                                break
                                        }
                                }
                                if win {
                                        return player, false
                                }
                        }
                }
        }

        for row := 0; row <= Rows-WinLength; row++ {
                for col := 0; col <= Cols-WinLength; col++ {
                        player := board[row][col]
                        if player != Empty {
                                win := true
                                for i := 1; i < WinLength; i++ {
                                        if board[row+i][col+i] != player {
                                                win = false
                                                break
                                        }
                                }
                                if win {
                                        return player, false
                                }
                        }
                }
        }

        for row := 0; row <= Rows-WinLength; row++ {
                for col := WinLength - 1; col < Cols; col++ {
                        player := board[row][col]
                        if player != Empty {
                                win := true
                                for i := 1; i < WinLength; i++ {
                                        if board[row+i][col-i] != player {
                                                win = false
                                                break
                                        }
                                }
                                if win {
                                        return player, false
                                }
                        }
                }
        }

        isFull := true
        for col := 0; col < Cols; col++ {
                if board[0][col] == Empty {
                        isFull = false
                        break
                }
        }

        if isFull {
                return Empty, true
        }

        return Empty, false
}

func IsValidMove(board *Board, column int) bool {
        if column < 0 || column >= Cols {
                return false
        }
        return board[0][column] == Empty
}

func GetValidColumns(board *Board) []int {
        valid := []int{}
        for col := 0; col < Cols; col++ {
                if board[0][col] == Empty {
                        valid = append(valid, col)
                }
        }
        return valid
}
