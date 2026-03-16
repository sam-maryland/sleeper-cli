package sleeper

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
)

const baseURL = "https://api.sleeper.app/v1"

// ISleeperClient is the interface for interacting with the Sleeper API.
type ISleeperClient interface {
	GetUser(ctx context.Context, userID string) (SleeperUser, error)
	GetLeague(ctx context.Context, leagueID string) (SleeperLeague, error)
	GetUsersInLeague(ctx context.Context, leagueID string) (SleeperUsers, error)
	GetRostersInLeague(ctx context.Context, leagueID string) (Rosters, error)
	GetMatchupsForWeek(ctx context.Context, leagueID string, week int) (Matchups, error)
	GetNFLState(ctx context.Context) (NFLState, error)
	GetWinnersBracket(ctx context.Context, leagueID string) ([]BracketMatchup, error)
	FetchAllPlayers(ctx context.Context) ([]byte, error)
}

type SleeperClient struct {
	httpClient *http.Client
}

func NewSleeperClient(c *http.Client) *SleeperClient {
	return &SleeperClient{httpClient: c}
}

func (c *SleeperClient) GetUser(ctx context.Context, userID string) (SleeperUser, error) {
	u := fmt.Sprintf("%s/user/%s", baseURL, userID)
	req, err := newJSONRequest(ctx, http.MethodGet, u, nil)
	if err != nil {
		return SleeperUser{}, err
	}
	res, err := c.httpClient.Do(req)
	var user SleeperUser
	if err := jsonResponder(res, err, &user); err != nil {
		return SleeperUser{}, err
	}
	return user, nil
}

func (c *SleeperClient) GetLeague(ctx context.Context, leagueID string) (SleeperLeague, error) {
	u := fmt.Sprintf("%s/league/%s", baseURL, leagueID)
	req, err := newJSONRequest(ctx, http.MethodGet, u, nil)
	if err != nil {
		return SleeperLeague{}, err
	}
	res, err := c.httpClient.Do(req)
	var league SleeperLeague
	if err := jsonResponder(res, err, &league); err != nil {
		return SleeperLeague{}, err
	}
	return league, nil
}

func (c *SleeperClient) GetUsersInLeague(ctx context.Context, leagueID string) (SleeperUsers, error) {
	u := fmt.Sprintf("%s/league/%s/users", baseURL, leagueID)
	req, err := newJSONRequest(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	res, err := c.httpClient.Do(req)
	var users SleeperUsers
	if err := jsonResponder(res, err, &users); err != nil {
		return nil, err
	}
	return users, nil
}

func (c *SleeperClient) GetRostersInLeague(ctx context.Context, leagueID string) (Rosters, error) {
	u := fmt.Sprintf("%s/league/%s/rosters", baseURL, leagueID)
	req, err := newJSONRequest(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	res, err := c.httpClient.Do(req)
	var rosters Rosters
	if err := jsonResponder(res, err, &rosters); err != nil {
		return nil, err
	}
	return rosters, nil
}

func (c *SleeperClient) GetMatchupsForWeek(ctx context.Context, leagueID string, week int) (Matchups, error) {
	u := fmt.Sprintf("%s/league/%s/matchups/%d", baseURL, leagueID, week)
	req, err := newJSONRequest(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	res, err := c.httpClient.Do(req)
	var matchups Matchups
	if err := jsonResponder(res, err, &matchups); err != nil {
		return nil, err
	}
	return matchups, nil
}

func (c *SleeperClient) GetNFLState(ctx context.Context) (NFLState, error) {
	u := fmt.Sprintf("%s/state/nfl", baseURL)
	req, err := newJSONRequest(ctx, http.MethodGet, u, nil)
	if err != nil {
		return NFLState{}, err
	}
	res, err := c.httpClient.Do(req)
	var state NFLState
	if err := jsonResponder(res, err, &state); err != nil {
		return NFLState{}, err
	}
	return state, nil
}

func (c *SleeperClient) GetWinnersBracket(ctx context.Context, leagueID string) ([]BracketMatchup, error) {
	u := fmt.Sprintf("%s/league/%s/winners_bracket", baseURL, leagueID)
	req, err := newJSONRequest(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	res, err := c.httpClient.Do(req)
	var bracket []BracketMatchup
	if err := jsonResponder(res, err, &bracket); err != nil {
		return nil, err
	}
	return bracket, nil
}

func (c *SleeperClient) FetchAllPlayers(ctx context.Context) ([]byte, error) {
	u := fmt.Sprintf("%s/players/nfl", baseURL)
	req, err := newJSONRequest(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	res, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := res.Body.Close(); err != nil {
			log.Printf("error closing response body: %v", err)
		}
	}()
	return io.ReadAll(res.Body)
}

// --- unexported HTTP helpers ---

func newJSONRequest(ctx context.Context, method, url string, body any) (*http.Request, error) {
	b := new(bytes.Buffer)
	if body != nil {
		if err := json.NewEncoder(b).Encode(body); err != nil {
			return nil, err
		}
	}
	req, err := http.NewRequestWithContext(ctx, method, url, b)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	return req, nil
}

func jsonResponder(res *http.Response, err error, v any) error {
	if err != nil {
		return err
	}
	if res == nil {
		return errors.New("response was nil")
	}
	defer func() {
		if err := res.Body.Close(); err != nil {
			log.Printf("error closing response body: %v", err)
		}
	}()
	if res.StatusCode >= 300 {
		return fmt.Errorf("sleeper API returned status %d", res.StatusCode)
	}
	if v == nil {
		return nil
	}
	return json.NewDecoder(res.Body).Decode(v)
}
