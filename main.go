package main

import (
	"flag"
	"fmt"
	"github.com/joho/godotenv"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
)

type EpisodeMatch struct {
	Filename      string
	EpisodeNumber int
	MatchType     string
	Found         bool
}

type RenamePlan struct {
	Original  string
	Proposed  string
	Episode   EpisodeMatch
	CanRename bool
	Reason    string
}

func main() {
	// Flags
	seriesPtr := flag.String("series", "", "string representing series title")
	flag.Parse()

	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	apiKey := os.Getenv("TVDB_API_KEY")
	if apiKey == "" {
		log.Fatal("TVDB_API_KEY not set")
	}

	client := NewTVDBClient(apiKey)

	err = client.Login()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("renamer started...")

	if *seriesPtr == "" {
		fmt.Println("ERROR: --series flag is required")
		flag.Usage()
		os.Exit(1)
	}
	fmt.Println("Series Title: ", *seriesPtr)
	entries, err := os.ReadDir(".")

	if err != nil {
		log.Fatal(err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			if filepath.Ext(entry.Name()) == ".mkv" {
				episode := matchEpisodeNumber(entry.Name())
				fmt.Println("Filename: ", episode.Filename)
				fmt.Println("Num: ", episode.EpisodeNumber)
				fmt.Println("Type: ", episode.MatchType)
				fmt.Println("Found?: ", episode.Found)
			}
		}
	}
}

func matchEpisodeNumber(filename string) EpisodeMatch {
	strictExp := regexp.MustCompile(`(?:\s|-)(\d{1,4})(?:\s|-)`)
	looseExp := regexp.MustCompile(`\b\d{1,4}\b`)

	// Try strict match first
	if strictMatch := strictExp.FindStringSubmatch(filename); len(strictMatch) >= 2 {
		ep, err := strconv.Atoi(strictMatch[1])
		if err == nil {
			return EpisodeMatch{
				Filename:      filename,
				EpisodeNumber: ep,
				MatchType:     "strict",
				Found:         true,
			}
		}
	}

	// Fallback to loose matching
	looseMatches := looseExp.FindAllString(filename, -1)
	var bestMatch int

	for _, match := range looseMatches {
		num, err := strconv.Atoi(match)
		if err != nil {
			continue
		}

		if num >= 1900 {
			continue
		}
		if num == 720 || num == 1080 || num == 2160 {
			continue
		}

		bestMatch = num // last valid wins
	}

	if bestMatch != 0 {
		return EpisodeMatch{
			Filename:      filename,
			EpisodeNumber: bestMatch,
			MatchType:     "loose",
			Found:         true,
		}
	}

	//  Nothing matched
	return EpisodeMatch{
		Filename: filename,
		Found:    false,
	}
}
