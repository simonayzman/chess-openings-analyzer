package fetcher

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// ChessOpeningGames defines a set of professionally rated games
// using one opening, found on http://www.pgnmentor.com/
type ChessOpeningGames struct {
	Name     string
	URL      string
	Moves    string
	Games    []string
	NumGames int
}

// readZipFile is a small helper to parse zip files
// https://stackoverflow.com/questions/50539118/golang-unzip-response-body/50539327
func readZipFile(zf *zip.File) ([]byte, error) {
	f, err := zf.Open()
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return ioutil.ReadAll(f)
}

// Fetch scrapes www.pgnmentor.com for chess games, centered around a large
// number of chess openings. It stores all of these games under data/all_games
func Fetch() {
	workingDirectory, _ := os.Getwd()
	allGamesPath := filepath.Join(workingDirectory, "../data/all_games/")
	os.MkdirAll(allGamesPath, 0755)

	baseURL := "http://www.pgnmentor.com/"
	homePageURL := baseURL + "files.html"

	// Request the HTML page.
	res, err := http.Get(homePageURL)
	if err != nil {
		log.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		log.Fatalf("status code error: %d %s", res.StatusCode, res.Status)
	}

	// Load the HTML document
	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		log.Fatal(err)
	}

	var allGames []ChessOpeningGames
	doc.Find("#openings").NextUntil("#events").Filter("table[border='3']").Each(func(tableId int, tableSelection *goquery.Selection) {
		tableSelection.Find("tr").Each(func(rowId int, rowSelection *goquery.Selection) {
			chessGames := ChessOpeningGames{}

			rowSelection.Find("td").Each(func(dataId int, dataSelection *goquery.Selection) {
				if dataId == 0 {
					chessGames.URL, _ = dataSelection.Find("a").First().Attr("href")
				} else if dataId == 1 {
					contents := dataSelection.Contents()
					chessGames.Name = strings.TrimSpace(contents.First().Text())
					numGamesString := strings.Replace(contents.Last().Text(), " games", "", 1)
					chessGames.NumGames, _ = strconv.Atoi(numGamesString)
				} else if dataId == 2 {
					moves := ""
					dataSelection.Contents().Each(func(moveId int, moveSelection *goquery.Selection) {
						if moveSelection.Is("img") {
							piece, _ := moveSelection.Attr("src")
							pieceLetter := strings.ToUpper(string(piece[1]))
							moves = moves + pieceLetter
						} else {
							moves = moves + moveSelection.Text()
						}
					})
					chessGames.Moves = strings.TrimSpace(moves)
				}
			})
			allGames = append(allGames, chessGames)
		})
	})

	for _, openingGames := range allGames {
		openingGamesURL := baseURL + openingGames.URL
		res, _ := http.Get(openingGamesURL)

		body, err := ioutil.ReadAll(res.Body)
		if err != nil {
			log.Fatal(err)
		}

		zipReader, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
		if err != nil {
			log.Fatal(err)
		}

		for _, zipFile := range zipReader.File {
			fmt.Println("Reading file:", zipFile.Name)
			unzippedFileBytes, err := readZipFile(zipFile)

			if err != nil {
				log.Println(err)
				continue
			}

			filePath := filepath.Join(workingDirectory, "../data/all_games/", zipFile.Name)
			outFile, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, zipFile.Mode())
			outFile.Write(unzippedFileBytes)
			outFile.Close()
		}
	}
}
