package processor

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"

	"github.com/cornelk/hashmap"

	"github.com/notnil/chess"
)

type processorContext struct {
	NumThreads           int
	MovesDepth           int
	PositionOutcomeIndex *hashmap.HashMap
}

type processorTask struct {
	PGNFilePath string
}

type processorResult struct {
	IndexKeyToIncrement string
}

func getPositionOutcomeIndexKey(position *chess.Position, outcome chess.Outcome) string {
	var keyBuilder strings.Builder
	keyBuilder.WriteString(position.Board().String())
	keyBuilder.WriteString(" ")
	keyBuilder.WriteString(position.Turn().String())
	keyBuilder.WriteString(" ")
	keyBuilder.WriteString(position.CastleRights().String())
	keyBuilder.WriteString(" ")
	keyBuilder.WriteString(outcome.String())
	return keyBuilder.String()
}

func incrementPositionOutcome(context *processorContext, indexKey string) {
	var i int64
	actual, _ := context.PositionOutcomeIndex.GetOrInsert(indexKey, &i)
	counter := (actual).(*int64)
	atomic.AddInt64(counter, 1)
}

func savePositionOutcomeIndex(context *processorContext) {
	// Convert index into JSON
	analysisMap := make(map[string]interface{})
	for entry := range context.PositionOutcomeIndex.Iter() {
		valueAddress := (entry.Value).(*int64)
		key := (entry.Key).(string)
		value := *valueAddress
		analysisMap[key] = value
	}
	analysisJSON, _ := json.Marshal(analysisMap)
	analysisJSONString := string(analysisJSON)

	// Save analysis JSON
	workingDirectory, _ := os.Getwd()
	analysisFilePath := filepath.Join(workingDirectory, "../data/analysis.json")
	analysisFile, _ := os.OpenFile(analysisFilePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 666)
	analysisFile.WriteString(analysisJSONString)
	analysisFile.Close()
}

func processSequential(context *processorContext) {
	// Cycle through all pgn files and build index
	workingDirectory, _ := os.Getwd()
	sampledGamesPath := filepath.Join(workingDirectory, "../data/sampled_games/")
	filepath.Walk(sampledGamesPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			log.Fatalf(err.Error())
		}
		fmt.Printf("File Name: %s\n", info.Name())

		if info.IsDir() {
			return nil
		}

		gameFile, _ := os.Open(path)
		games, _ := chess.GamesFromPGN(gameFile)

		for _, game := range games {
			outcome := game.Outcome()
			if outcome != chess.WhiteWon && outcome != chess.BlackWon && outcome != chess.Draw {
				break
			}

			positionDepth := context.MovesDepth * 2 // how deep into the game to build analysis index
			for positionCount, position := range game.Positions() {
				if positionCount >= positionDepth {
					break
				}

				indexKey := getPositionOutcomeIndexKey(position, outcome)
				incrementPositionOutcome(context, indexKey)
			}
		}

		return nil
	})

	savePositionOutcomeIndex(context)
}

func processParallel(context *processorContext) {
	done := make(chan bool)
	defer close(done)

	processorTaskStream := getProcessorTaskStream(done, context)

	workers := make([]<-chan processorResult, context.NumThreads)
	for i := 0; i < context.NumThreads; i++ {
		workers[i] = getProcessorResultStream(done, processorTaskStream, context)
	}

	for processorResult := range getMergedProcessorResultStream(done, workers...) {
		incrementPositionOutcome(context, processorResult.IndexKeyToIncrement)
	}

	savePositionOutcomeIndex(context)
}

func getProcessorTaskStream(
	done <-chan bool,
	context *processorContext,
) <-chan processorTask {
	processorTaskStream := make(chan processorTask)
	go func() {
		defer close(processorTaskStream)

		workingDirectory, _ := os.Getwd()
		sampledGamesPath := filepath.Join(workingDirectory, "../data/sampled_games/")
		filepath.Walk(sampledGamesPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				log.Fatalf(err.Error())
			}
			fmt.Printf("File Name: %s\n", info.Name())

			if info.IsDir() {
				return nil
			}

			task := processorTask{path}

			select {
			case <-done:
				return nil
			case processorTaskStream <- task:
				// fmt.Println("Pushing processor task for", task)
			}

			return nil
		})
	}()
	return processorTaskStream
}

func getProcessorResultStream(
	done <-chan bool,
	processorTaskStream <-chan processorTask,
	context *processorContext,
) <-chan processorResult {
	processorResultStream := make(chan processorResult)

	go func() {
		defer close(processorResultStream)
		for task := range processorTaskStream {
			// fmt.Println("Pulling processor task for", task)

			gameFile, _ := os.Open(task.PGNFilePath)
			games, _ := chess.GamesFromPGN(gameFile)

			for _, game := range games {
				outcome := game.Outcome()
				if outcome != chess.WhiteWon && outcome != chess.BlackWon && outcome != chess.Draw {
					break
				}

				positionDepth := context.MovesDepth * 2 // how deep into the game to build analysis index
				for positionCount, position := range game.Positions() {
					if positionCount >= positionDepth {
						break
					}

					indexKey := getPositionOutcomeIndexKey(position, outcome)
					result := processorResult{indexKey}

					select {
					case <-done:
						return
					case processorResultStream <- result:
						// fmt.Println("Pushing processor result for", result)
					}
				}
			}
		}
	}()

	return processorResultStream
}

func getMergedProcessorResultStream(
	done <-chan bool,
	processorResultStreams ...<-chan processorResult,
) <-chan processorResult {
	waitGroupChannel := make(chan bool, len(processorResultStreams))
	resultStreamsLeft := len(processorResultStreams)

	multiplexedProcessorResultStream := make(chan processorResult)
	multiplex := func(processorResultStream <-chan processorResult) {
		for result := range processorResultStream {
			select {
			case <-done:
				waitGroupChannel <- true
				return
			case multiplexedProcessorResultStream <- result:
				// fmt.Println("Pushing multiplexed processor result for", result)
			}
		}
		waitGroupChannel <- true
	}

	for _, processorResultStream := range processorResultStreams {
		go multiplex(processorResultStream)
	}

	go func() {
		for range waitGroupChannel {
			resultStreamsLeft--
			if resultStreamsLeft == 0 {
				break
			}
		}
		close(multiplexedProcessorResultStream)
	}()

	return multiplexedProcessorResultStream
}

// Process accepts a mode for invocation (sequential or parallel), and if parallel,
// the number of threads to run the program with. It then builds an index, i.e.
// a hashmap, where the keys are board positions (using the FEN format) + outcomes
// (white wins, black wins, draw), and the values are how many historical games
// reached that position + outcome combination. This index is then saved as a
// JSON string and later used for analysis.
func Process(parallelMode bool, numThreads int) {
	var context processorContext
	context.NumThreads = numThreads
	context.MovesDepth = 10
	context.PositionOutcomeIndex = hashmap.New(10000)

	if parallelMode {
		runtime.GOMAXPROCS(context.NumThreads)
		processParallel(&context)
	} else {
		processSequential(&context)
	}
}
