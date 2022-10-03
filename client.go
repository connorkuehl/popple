package popple

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

type Client struct {
	client  http.Client
	baseURL string
}

func NewHTTPClient(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
	}
}

func (c *Client) Board(ctx context.Context, serverID string, ord BoardOrder, limit uint) (Board, error) {
	url := fmt.Sprintf("%s/boards/%s", c.baseURL, serverID)

	var args struct {
		Order string `json:"order"`
		Limit uint   `json:"limit"`
	}

	args.Limit = limit
	switch ord {
	case BoardOrderAsc:
		args.Order = "asc"
	case BoardOrderDsc:
		args.Order = "desc"
	}

	payload, err := json.Marshal(args)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}

	rsp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer rsp.Body.Close()

	var board Board
	err = json.NewDecoder(rsp.Body).Decode(&board)
	if err != nil {
		return nil, err
	}

	return board, nil
}

func (c *Client) ChangeKarma(ctx context.Context, serverID string, increments Increments) (Increments, error) {
	url := fmt.Sprintf("%s/counts/%s", c.baseURL, serverID)

	payload, err := json.Marshal(increments)
	if err != nil {
		return nil, err
	}

	rsp, err := c.client.Post(url, "application/json", bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	defer rsp.Body.Close()

	var newLevels Increments
	err = json.NewDecoder(rsp.Body).Decode(&newLevels)
	if err != nil {
		return nil, err
	}

	return newLevels, nil
}

func (c *Client) CheckKarma(ctx context.Context, serverID string, who []string) (map[string]int64, error) {
	url := fmt.Sprintf("%s/counts/%s", c.baseURL, serverID)

	payload, err := json.Marshal(who)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}

	rsp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer rsp.Body.Close()

	var levels map[string]int64
	err = json.NewDecoder(rsp.Body).Decode(&levels)
	if err != nil {
		return nil, err
	}

	return levels, nil
}

func (c *Client) Config(ctx context.Context, serverID string) (*Config, error) {
	url := fmt.Sprintf("%s/configs/%s", c.baseURL, serverID)

	rsp, err := c.client.Get(url)
	if err != nil {
		return nil, err
	}
	defer rsp.Body.Close()

	var config Config
	err = json.NewDecoder(rsp.Body).Decode(&config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}

func (c *Client) PutConfig(ctx context.Context, config *Config) error {
	url := fmt.Sprintf("%s/configs/%s", c.baseURL, config.ServerID)

	payload, err := json.Marshal(config)
	if err != nil {
		return err
	}

	rsp, err := c.client.Post(url, "application/json", bytes.NewReader(payload))
	if err != nil {
		return err
	}
	defer rsp.Body.Close()

	return err
}
