package clients

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/maxmorhardt/squares-api/internal/model"
	"github.com/maxmorhardt/squares-api/internal/util"
)

const scoreboardPath = "/apis/site/v2/sports/football/nfl/scoreboard"

// identify ourselves rather than sending resty's default
const userAgent = "squares-api (+https://github.com/maxmorhardt/squares-api)"

type ESPNClient interface {
	FetchScoreboard(ctx context.Context, dates []string) ([]model.ESPNGame, error)
}

type espnClient struct {
	client *resty.Client
}

func NewESPNClient(baseURL string) ESPNClient {
	// the poll interval is longer than a pooled connection stays good, so retire idle ones early
	transport := &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		DialContext:           (&net.Dialer{Timeout: 5 * time.Second, KeepAlive: 30 * time.Second}).DialContext,
		MaxIdleConnsPerHost:   2,
		IdleConnTimeout:       30 * time.Second,
		TLSHandshakeTimeout:   5 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	return &espnClient{
		client: resty.New().
			SetBaseURL(baseURL).
			SetTransport(transport).
			SetHeader("User-Agent", userAgent).
			SetTimeout(10 * time.Second).
			SetRetryCount(1).
			SetRetryWaitTime(1 * time.Second).
			SetRetryMaxWaitTime(3 * time.Second),
	}
}

func (c *espnClient) FetchScoreboard(ctx context.Context, dates []string) ([]model.ESPNGame, error) {
	// no dates at all falls back to espn's default, which is the current week's slate
	if len(dates) == 0 {
		return c.fetchOne(ctx, "")
	}

	// a game near the day boundary comes back under both adjacent dates, so merge on espn id
	seen := make(map[string]int, len(dates)*16)
	merged := make([]model.ESPNGame, 0, len(dates)*16)
	for _, date := range dates {
		games, err := c.fetchOne(ctx, date)
		if err != nil {
			return nil, err
		}

		for i := range games {
			if j, ok := seen[games[i].ESPNID]; ok {
				merged[j] = games[i]
				continue
			}
			seen[games[i].ESPNID] = len(merged)
			merged = append(merged, games[i])
		}
	}

	return merged, nil
}

func (c *espnClient) fetchOne(ctx context.Context, date string) ([]model.ESPNGame, error) {
	req := c.client.R().
		SetContext(ctx).
		SetQueryParam("limit", "100").
		ForceContentType("application/json").
		SetResult(&model.ScoreboardResponse{})
	if date != "" {
		req.SetQueryParam("dates", date)
	}

	resp, err := req.Get(scoreboardPath)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch scoreboard: %w", err)
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("scoreboard returned status %d for dates %q", resp.StatusCode(), date)
	}

	body, ok := resp.Result().(*model.ScoreboardResponse)
	if !ok {
		return nil, fmt.Errorf("unexpected scoreboard response type")
	}

	return util.ScoreboardToGames(body), nil
}
