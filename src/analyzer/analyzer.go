package analyzer

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"github.com/notnil/chess"
)

// Analyze takes in input type (of the form 'pgn' or 'fen') and an input file
// (path) that matches the provided type. Using the provided chess game position,
// it looks at (an indexed version of) historical games that have reached this
// position and their outcomes (white won / black won / draw), and spits out
// the breakdowns in terms of percents and absolute values.
func Analyze(inputType string, inputFile string) {
	var game *chess.Game
	if inputType == "pgn" {
		pgnFile, openError := os.Open(inputFile)
		if openError != nil {
			log.Fatalln("Failed to open PGN file")
		}

		pgn, pgnError := chess.PGN(pgnFile)
		if pgnError != nil {
			log.Fatalln("Invalid PGN file. You may have accidentally provided a FEN file, or the PGN file contains multiple games.")
		}

		game = chess.NewGame(pgn)
	} else if inputType == "fen" {
		fenFile, openError := os.Open(inputFile)
		if openError != nil {
			log.Fatalln("Failed to open FEN file")
		}

		fenBytes, _ := ioutil.ReadAll(fenFile)
		fen, fenError := chess.FEN(string(fenBytes))
		if fenError != nil {
			log.Fatalln("Invalid FEN file. You may have accidentally provided a PGN file.")
		}

		game = chess.NewGame(fen)
	} else {
		log.Fatalln("Input type invalid. Must either be 'pgn' or 'fen'.")
	}

	// Set up game
	position := game.Position()
	board := position.Board().String()
	turn := position.Turn().String()
	castling := position.CastleRights().String()

	baseKey := board + " " + turn + " " + castling + " "
	whiteWonKey := baseKey + chess.WhiteWon.String()
	blackWonKey := baseKey + chess.BlackWon.String()
	drawKey := baseKey + chess.Draw.String()

	// Open analysis
	workingDirectory, _ := os.Getwd()
	analysisPath := filepath.Join(workingDirectory, "../data/analysis.json")

	jsonFile, err := os.Open(analysisPath)
	if err != nil {
		log.Fatalln(err)
	}

	byteValue, _ := ioutil.ReadAll(jsonFile)
	var analysis map[string]int64
	json.Unmarshal([]byte(byteValue), &analysis)
	jsonFile.Close()

	// Perform analysis
	var totalGames int64

	whiteWonCount, whiteWonFound := analysis[whiteWonKey]
	if whiteWonFound {
		totalGames += whiteWonCount
	}

	blackWonCount, blackWonFound := analysis[blackWonKey]
	if blackWonFound {
		totalGames += blackWonCount
	}

	drawCount, drawFound := analysis[drawKey]
	if drawFound {
		totalGames += drawCount
	}

	fmt.Println(game.Position().Board().Draw())

	if totalGames == 0 {
		fmt.Println("No games found with this position.")
		return
	}

	whiteWonPercent := float64(whiteWonCount) / float64(totalGames) * 100
	blackWonPercent := float64(blackWonCount) / float64(totalGames) * 100
	drawPercent := float64(drawCount) / float64(totalGames) * 100

	fmt.Printf("Historically, in this position (%s to move)\n", game.Position().Turn().Name())
	fmt.Println("==============================================")
	fmt.Printf("White WON:\t%.1f%% (%d games)\n", whiteWonPercent, whiteWonCount)
	fmt.Printf("Black WON:\t%.1f%% (%d games)\n", blackWonPercent, blackWonCount)
	fmt.Printf("DRAWN:\t\t%.1f%% (%d games) \n", drawPercent, drawCount)
}
