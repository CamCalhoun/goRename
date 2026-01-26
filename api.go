package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/charmbracelet/huh"
	"net/http"
	"net/url"
	"strconv"
)

type TVDBClient struct {
	apiKey string
	token  string
	client *http.Client
}

type Series struct {
	TVDBID       string            `json:"tvdb_id"`
	Name         string            `json:"name"`
	Year         string            `json:"year"`
	Type         string            `json:"type"`
	Translations map[string]string `json:"translations"`
}

type Episode struct {
	ID            int      `json:"id"`
	Name          string   `json:"name"`
	SeasonNumber  int      `json:"seasonNumber"`
	Number        int      `json:"number"`
	NameLanguages []string `json:"nameTranslations"`
}

func NewTVDBClient(apiKey string) *TVDBClient {
	return &TVDBClient{
		apiKey: apiKey,
		client: &http.Client{},
	}
}

func (c *TVDBClient) Login() error {
	type loginRequest struct {
		APIKey string `json:"apikey"`
	}

	type loginResponse struct {
		Status string `json:"status"`
		Data   struct {
			Token string `json:"token"`
		} `json:"data"`
	}

	payload := loginRequest{
		APIKey: c.apiKey,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	resp, err := c.client.Post("https://api4.thetvdb.com/v4/login", "application/json", bytes.NewReader(body))
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	var loginResp loginResponse

	err = json.NewDecoder(resp.Body).Decode(&loginResp)
	if err != nil {
		return err
	}

	c.token = loginResp.Data.Token
	return nil
}

func (c *TVDBClient) searchSeries(seriesName string) (*Series, error) {
	type seriesResponse struct {
		Status string   `json:"status"`
		Data   []Series `json:"data"`
	}

	// Build base URL
	baseURL := "https://api4.thetvdb.com/v4/search"
	u, err := url.Parse(baseURL)
	if err != nil {
		return nil, err
	}
	// Build query
	q := u.Query()
	q.Set("query", seriesName)
	q.Set("type", "series")
	u.RawQuery = q.Encode()
	// Assemble request
	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		return nil, err
	}
	// Attach headers
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/json")
	// Do request
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Store response
	var seriesResp seriesResponse
	err = json.NewDecoder(resp.Body).Decode(&seriesResp)
	if err != nil {
		return nil, err
	}

	// Grab english name, set this to Name
	// If only one response, return it
	if len(seriesResp.Data) == 1 {
		return &seriesResp.Data[0], nil
	}

	// User selection if more than one response
	var selectedIndex int
	options := make([]huh.Option[int], 0, len(seriesResp.Data))

	for i := range seriesResp.Data {
		if eng, ok := seriesResp.Data[i].Translations["eng"]; ok && eng != "" {
			seriesResp.Data[i].Name = eng
		}

		s := seriesResp.Data[i]
		label := fmt.Sprintf("%s %s", s.Name, s.Year)
		options = append(options, huh.NewOption(label, i))
	}

	err = huh.NewSelect[int]().
		Title("Select the correct series").
		Description("Use ↑ ↓ to navigate • Enter to confirm").
		Options(options...).
		Value(&selectedIndex).
		Run()

	if err != nil {
		return nil, err
	}

	selectedSeries := seriesResp.Data[selectedIndex]
	return &selectedSeries, err
}

func (c *TVDBClient) searchEpisode(series *Series, episodeMatch *EpisodeMatch) (*Episode, error) {
	type episodeResponse struct {
		Status string `json:"status"`
		Data   struct {
			Episodes []Episode `json:"episodes"`
		} `json:"data"`
	}

	// Build base URL

	baseURL := "https://api4.thetvdb.com/v4/series/" + series.TVDBID + "/episodes/absolute"
	u, err := url.Parse(baseURL)
	if err != nil {
		return nil, err
	}

	// Build query
	q := u.Query()
	q.Set("page", "1")
	q.Set("season", "1")
	q.Set("episodeNumber", strconv.Itoa(episodeMatch.EpisodeNumber))
	u.RawQuery = q.Encode()

	// Assemble request
	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		return nil, err
	}

	// Attach headers

	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/json")
	// Do request
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Store response
	var episodeResp episodeResponse
	err = json.NewDecoder(resp.Body).Decode(&episodeResp)
	if err != nil {
		return nil, err
	}

	if len(episodeResp.Data.Episodes) == 0 {
		err = errors.New("No episodes found...")
		return nil, err
	}
	episode := episodeResp.Data.Episodes[0]

	return &episode, err
}

func (c *TVDBClient) grabEpisodeInfo(episode *Episode) (*RenamePlan, error) {
	type episodeInfoResponse struct {
		Status string `json:"status"`
		Data   struct {
			ID           int `json:"id"`
			Translations struct {
				NameTranslations []struct {
					Name     string `json:"name"`
					Language string `json:"language"`
				} `json:"nameTranslations"`
			} `json:"translations"`
			Number  int `json:"number"`
			Seasons []struct {
				Number int `json:"number"`
			} `json:"seasons"`
		} `json:"data"`
	}

	// Base URL
	baseURL := "https://api4.thetvdb.com/v4/episodes/" + strconv.Itoa(episode.ID) + "/extended"
	u, err := url.Parse(baseURL)
	if err != nil {
		return nil, err
	}

	// Build Query
	q := u.Query()
	q.Set("meta", "translations")
	u.RawQuery = q.Encode()

	// Assemble request
	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		return nil, err
	}

	// Attach headers
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/json")
	// Do request
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var episodeInfoResp episodeInfoResponse
	err = json.NewDecoder(resp.Body).Decode(&episodeInfoResp)
	if err != nil {
		return nil, err
	}

	var englishTitle string
	for _, t := range episodeInfoResp.Data.Translations.NameTranslations {
		if t.Language == "eng" {
			englishTitle = t.Name
			break
		}
	}

	var renamePlan RenamePlan

	renamePlan.SeasonalEpisodeNumber = episodeInfoResp.Data.Number
	renamePlan.SeasonNumber = episodeInfoResp.Data.Seasons[0].Number
	renamePlan.TVDBName = englishTitle

	return &renamePlan, err
}
