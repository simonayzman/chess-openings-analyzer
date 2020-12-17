package sampler

import (
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
)

// Sample accepts a sampling rate parameter (between 0 and 1.0 inclusive), and takes
// the chess games found in 'data/all_games', samples a portion of each file based
// on the sampling rate, and saves the sampled version under 'data/sampled_games'
func Sample(samplingRate float64) {
	rand.Seed(721) // Lucky number!

	workingDirectory, _ := os.Getwd()
	allGamesPath := filepath.Join(workingDirectory, "../data/all_games/")
	filepath.Walk(allGamesPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			log.Fatalf(err.Error())
		}
		fmt.Printf("File Name: %s\n", info.Name())

		if info.IsDir() {
			return nil
		}

		sampledFilePath := filepath.Join(workingDirectory, "../data/sampled_games/", info.Name())
		sampledFile, err := os.OpenFile(sampledFilePath,
			os.O_WRONLY|os.O_CREATE|os.O_TRUNC,
			info.Mode())

		originalFile, _ := os.Open(path)
		originalFileBytes, _ := ioutil.ReadAll(originalFile)

		allPGNs := string(originalFileBytes)
		markedPGNS := strings.ReplaceAll(allPGNs, "[Event", "XXXXX[Event")
		splitPGNS := strings.Split(markedPGNS, "XXXXX")

		for _, pgn := range splitPGNS {
			currentRandomNumber := rand.Float64()
			if currentRandomNumber < samplingRate { // allow only a portion of games through
				sampledFile.WriteString(pgn)
			}
		}

		sampledFile.Close()
		originalFile.Close()

		return nil
	})
}
