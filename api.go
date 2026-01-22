package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

type TVDBClient struct {
	apiKey string
	token  string
	client *http.Client
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

func searchSeries() {}

func searchEpisode() {}
