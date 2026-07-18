package clients

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/maxmorhardt/squares-api/internal/model"
	"github.com/maxmorhardt/squares-api/internal/util"
)

const scoreboardPath = "/apis/site/v2/sports/football/nfl/scoreboard"

type ESPNClient interface {
	FetchScoreboard(ctx context.Context, dates string) ([]model.ESPNGame, error)
}

type espnClient struct {
	client *resty.Client
}

func NewESPNClient(baseURL string) ESPNClient {
	return &espnClient{
		client: resty.New().
			SetBaseURL(baseURL).
			SetTimeout(15 * time.Second).
			SetRetryCount(2).
			SetRetryWaitTime(200 * time.Millisecond),
	}
}

func (c *espnClient) FetchScoreboard(ctx context.Context, dates string) ([]model.ESPNGame, error) {
	req := c.client.R().
		SetContext(ctx).
		SetQueryParam("limit", "100").
		ForceContentType("application/json").
		SetResult(&model.ScoreboardResponse{})
	if dates != "" {
		req.SetQueryParam("dates", dates)
	}

	resp, err := req.Get(scoreboardPath)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch scoreboard: %w", err)
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("scoreboard returned status %d", resp.StatusCode())
	}

	body, ok := resp.Result().(*model.ScoreboardResponse)
	if !ok {
		return nil, fmt.Errorf("unexpected scoreboard response type")
	}

	return util.ScoreboardToGames(body), nil
}
