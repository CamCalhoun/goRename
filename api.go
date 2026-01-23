package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/charmbracelet/huh"
	"net/http"
	"net/url"
)

type TVDBClient struct {
	apiKey string
	token  string
	client *http.Client
}

type Series struct {
	TVDBID string `json:"tvdb_id"`
	Name   string `json:"name"`
	Year   string `json:"year"`
	Type   string `json:"type"`
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
	// If only one response, return it
	if len(seriesResp.Data) == 1 {
		return &seriesResp.Data[0], nil
	}

	// User selection if more than one response
	var selectedIndex int
	options := make([]huh.Option[int], 0, len(seriesResp.Data))

	for i, s := range seriesResp.Data {
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

func searchEpisode() {}
