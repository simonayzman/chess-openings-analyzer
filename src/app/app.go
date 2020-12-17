package main

import (
	"chess-openings/analyzer"
	"chess-openings/fetcher"
	"chess-openings/processor"
	"chess-openings/sampler"
	"fmt"
	"os"
	"strconv"
)

// printUsage prints the usage statement for the program
func printUsage() {
	usage := "Usage: app <command> [...arguments]" +
		"\n\n\tapp help" +
		"\n\t\tShows this usage menu." +

		"\n\n\tapp fetch" +
		"\n\t\tFetches all the chess games from www.pgnmentor.com (about 1.8 GB of data)" +
		"\n\t\tand saves them to 'data/all_games'." +

		"\n\n\tapp sample [sampling rate]" +
		"\n\t\tSamples the fetched chess game files based on the provided sampling rate" +
		"\n\t\tand saves them to 'data/sampled_games'." +
		"\n\n\t\t[sampling rate] is an optional argument representing the sampling rate." +
		"\n\t\t\tIf not provided, the program defaults to a sampling rate of 0.5%%." +
		"\n\n\t\t\tIf provided and valid (between 0 and 1 inclusive), the program uses that value." +

		"\n\n\tapp process [number of workers]" +
		"\n\t\tBuilds an index of the saved chess games and saves the analysis to 'data/analysis.json'." +
		"\n\t\tIt can operate in either sequential mode or parallel mode." +
		"\n\n\t\t[number of workers] is an optional argument representing the program's parallelism." +
		"\n\t\t\tIf not provided, the program defaults to operating in sequential mode." +
		"\n\n\t\t\tIf provided, the program operates in parallel mode, limiting the maximum number of" +
		"\n\t\t\tprocessors using runtime.GOMAXPROCS(...) and spawning that number of workers." +

		"\n\n\tapp analyze <input type> <input file>" +
		"\n\t\tAnalyzes the provided chess game position using the calculated index." +
		"\n\n\t\t<input type> is a required argument that can either be 'pgn' or 'fen'." +
		"\n\t\t<input file> is a required argument that represents a chess game position file matching <input type>."
	fmt.Printf(usage)
}

func main() {
	argsWithoutProg := os.Args[1:]
	if len(argsWithoutProg) == 0 {
		printUsage()
		return
	}

	command := argsWithoutProg[0]
	if command == "fetch" {
		fetcher.Fetch()
	} else if command == "sample" {
		defaultSampleRate := 0.005
		sampleRate := defaultSampleRate
		if len(argsWithoutProg) > 1 {
			sampleRateArgument := argsWithoutProg[1]
			sampleRate, _ = strconv.ParseFloat(sampleRateArgument, 64)
			if sampleRate < 0 || sampleRate > 1 {
				printUsage()
				return
			}
		}
		sampler.Sample(sampleRate)
	} else if command == "process" {
		parallelMode := false
		numThreads := 1

		if len(argsWithoutProg) > 1 {
			parallelMode = true
			numThreadsArgument := argsWithoutProg[1]
			numThreads, _ = strconv.Atoi(numThreadsArgument)
			if numThreads == 0 {
				printUsage()
				return
			}
		}

		processor.Process(parallelMode, numThreads)
	} else if command == "analyze" {
		if len(argsWithoutProg) < 3 {
			printUsage()
			return
		}

		inputType := argsWithoutProg[1]
		inputFile := argsWithoutProg[2]
		analyzer.Analyze(inputType, inputFile)
	} else {
		printUsage()
	}
}
