package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strconv"

	"github.com/charmbracelet/huh"
	"github.com/joho/godotenv"
)

type EpisodeMatch struct {
	Filename      string
	EpisodeNumber int
	MatchType     string
	Found         bool
}

type RenamePlan struct {
	TVDBName              string
	SeasonalEpisodeNumber int
	SeasonNumber          int
	NewFileName           string
	OldFileName           string
}

func main() {
	// Flags
	seriesPtr := flag.String("series", "", "string representing series title")
	dir := flag.String("dir", ".", "Directory with video files")
	flag.Parse()

	_ = godotenv.Load()

	apiKey := os.Getenv("TVDB_API_KEY")
	if apiKey == "" {
		log.Fatal("TVDB_API_KEY is not set (env var or .env)")
	}

	client := NewTVDBClient(apiKey)

	err := client.Login()
	if err != nil {
		printError(err)
		os.Exit(1)
	}

	if *seriesPtr == "" {
		fmt.Println(ErrorStyle.Render("Error: --series flag is required"))
		flag.Usage()
		os.Exit(1)
	}

	var series *Series
	series, err = client.searchSeries(*seriesPtr)
	if err != nil {
		printError(err)
		os.Exit(1)
	}
	fmt.Println()
	fmt.Println(
		HighlightStyle.Render(
			fmt.Sprintf(
				"Selected: %s %s",
				series.Name,
				series.Year,
			),
		),
	)
	entries, err := os.ReadDir(*dir)

	if err != nil {
		printError(err)
		os.Exit(1)
	}

	var renamePlans []RenamePlan
	for _, entry := range entries {
		if !entry.IsDir() {
			if filepath.Ext(entry.Name()) == ".mkv" {
				episodeMatch := matchEpisodeNumber(entry.Name())
				if episodeMatch.Found == true {
					episode, err := client.searchEpisode(series, &episodeMatch)
					if err != nil {
						printError(err)
						return
					}
					renamePlan, err := client.grabEpisodeInfo(episode)
					if err != nil {
						printError(err)
						return
					}
					renamePlan.OldFileName = episodeMatch.Filename
					renamePlans = append(renamePlans, *renamePlan)
				}
			}
		}
	}

	if len(renamePlans) == 0 {
		err = errors.New("No valid files available to rename")
		printError(err)
		return
	}
	for idx := range renamePlans {
		renamePlans[idx].NewFileName = renamePlans[idx].createNewFileName(series.Name)
	}

	const (
		newMax = 90
		oldMax = 80
	)
	selected := []int{}
	options := make([]huh.Option[int], 0, len(renamePlans))

	for i, rp := range renamePlans {
		label := fmt.Sprintf(
			"%s\n→ %s\n",
			truncate(rp.OldFileName, oldMax),
			truncate(rp.NewFileName, newMax),
		)
		options = append(
			options,
			huh.NewOption(label, i).Selected(true),
		)
	}

	err = huh.NewMultiSelect[int]().
		Title("Select files to rename").
		Description("Space to toggle selection ● Enter to rename selected files").
		Options(options...).
		Value(&selected).
		Run()

	if err != nil {
		printError(err)
		return
	}

	fmt.Println(TitleStyle.Render("Files to be renamed:\n"))

	for _, idx := range selected {
		rp := renamePlans[idx]

		fmt.Println(
			OldStyle.Render(rp.OldFileName),
			arrowStyle.Render(" → "),
			NewStyle.Render(rp.NewFileName),
		)
	}

	confirm := false
	err = huh.NewConfirm().
		Title("Proceed with renaming these files?").
		Affirmative("Yes, rename").
		Negative("Cancel").
		Value(&confirm).
		Run()

	if err != nil {
		printError(err)
		return
	}

	if !confirm {
		fmt.Println(ErrorStyle.Render("Rename cancelled, no files have been renamed."))
		return
	}
	successCount := 0
	for _, idx := range selected {
		rp := renamePlans[idx]

		oldPath := filepath.Join(*dir, rp.OldFileName)
		newPath := filepath.Join(*dir, rp.NewFileName)

		if err := os.Rename(oldPath, newPath); err != nil {
			fmt.Println(ErrorStyle.Render("✖  " + rp.OldFileName))
			printError(err)
			continue
		}

		successCount++
		fmt.Println(SuccessStyle.Render("✔  " + rp.NewFileName))
	}

	fmt.Println()
	fmt.Println(SuccessStyle.Render(
		fmt.Sprintf("✔  Successfully renamed %d file(s)", successCount),
	))
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

func (rp RenamePlan) createNewFileName(s string) string {
	return fmt.Sprintf("%s S%02dE%d - %s.mkv",
		s,
		rp.SeasonNumber,
		rp.SeasonalEpisodeNumber,
		rp.TVDBName,
	)
}

func printError(err error) {
	fmt.Println()
	fmt.Println(ErrorStyle.Render("Error: " + err.Error()))
}

func (rp RenamePlan) Label() string {
	return fmt.Sprintf(
		"%s → %s",
		rp.OldFileName,
		rp.NewFileName,
	)
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	if max <= 3 {
		return s[:max]
	}
	return s[:max-3] + "..."
}
