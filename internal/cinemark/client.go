// Package cinemark é um cliente da API pública (sem login) usada pelo site
// www.cinemark.com.br. Os endpoints estão documentados em docs/API.md.
package cinemark

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const DefaultBaseURL = "https://br-www-frontend-ext-prod.cinemark.com.br/bff-api"

type Client struct {
	BaseURL string
	HTTP    *http.Client
	Retries int
}

func New() *Client {
	return &Client{
		BaseURL: DefaultBaseURL,
		HTTP:    &http.Client{Timeout: 30 * time.Second},
		Retries: 3,
	}
}

type envelope struct {
	Success      bool            `json:"success"`
	MessageError string          `json:"messageError"`
	DataResult   json.RawMessage `json:"dataResult"`
}

func (c *Client) get(ctx context.Context, path string, q url.Values, out any) error {
	u := c.BaseURL + path
	if len(q) > 0 {
		u += "?" + q.Encode()
	}
	var lastErr error
	for attempt := 0; attempt <= c.Retries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(time.Duration(attempt) * 700 * time.Millisecond):
			}
		}
		retry, err := c.do(ctx, u, out)
		if err == nil {
			return nil
		}
		lastErr = err
		if !retry {
			break
		}
	}
	return lastErr
}

// do faz uma requisição; o bool indica se vale a pena tentar de novo.
func (c *Client) do(ctx context.Context, u string, out any) (bool, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return false, err
	}
	// A API só responde quando parece vir do site.
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0 Safari/537.36")
	req.Header.Set("Origin", "https://www.cinemark.com.br")
	req.Header.Set("Referer", "https://www.cinemark.com.br/")
	req.Header.Set("Accept", "application/json")
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return ctx.Err() == nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 32<<20))
	if err != nil {
		return true, err
	}
	if resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
		return true, fmt.Errorf("HTTP %d em %s (possível bloqueio/limite do site)", resp.StatusCode, u)
	}
	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("HTTP %d em %s", resp.StatusCode, u)
	}
	var env envelope
	if err := json.Unmarshal(body, &env); err != nil {
		return false, fmt.Errorf("resposta inesperada de %s: %w", u, err)
	}
	if !env.Success {
		return false, errors.New(strings.TrimSpace(env.MessageError))
	}
	if out == nil {
		return false, nil
	}
	return false, json.Unmarshal(env.DataResult, out)
}

type State struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Code string `json:"code"`
}

type City struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type Theater struct {
	ID    int    `json:"code"`
	Name  string `json:"name"`
	City  string `json:"city"`
	State string `json:"state"`
}

type Movie struct {
	ID   string `json:"id"`
	Slug string `json:"slug"`
	Name string `json:"name"`
}

func (c *Client) States(ctx context.Context) ([]State, error) {
	var out []State
	return out, c.get(ctx, "/v1/states", nil, &out)
}

func (c *Client) Cities(ctx context.Context, stateID int) ([]City, error) {
	var out []City
	return out, c.get(ctx, "/v1/cities", url.Values{"stateId": {strconv.Itoa(stateID)}}, &out)
}

func (c *Client) Theaters(ctx context.Context, cityID int) ([]Theater, error) {
	var out []Theater
	return out, c.get(ctx, "/v1/theaters", url.Values{"cityId": {strconv.Itoa(cityID)}}, &out)
}

// Movies devolve os filmes em cartaz e em pré-venda da cidade.
func (c *Client) Movies(ctx context.Context, cityID int) ([]Movie, error) {
	seen := map[string]bool{}
	var all []Movie
	for _, path := range []string{"/v1/movies/onDisplayByCity", "/v1/movies/preSaleByCity"} {
		var page []Movie
		q := url.Values{"cityId": {strconv.Itoa(cityID)}, "pageNumber": {"1"}, "pageSize": {"200"}}
		if err := c.get(ctx, path, q, &page); err != nil {
			return nil, err
		}
		for _, m := range page {
			if !seen[m.ID] {
				seen[m.ID] = true
				all = append(all, m)
			}
		}
	}
	return all, nil
}

// TheaterDay agrupa as sessões de um cinema em uma data.
type TheaterDay struct {
	TheaterID   int    `json:"theaterId"`
	TheaterName string `json:"theaterName"`
	Date        string `json:"date"`
	Rooms       []Room `json:"rooms"`
}

type Room struct {
	Number   int       `json:"number"`
	Features []int     `json:"features"`
	Audio    int       `json:"audio"`
	Sessions []Session `json:"sessions"`
}

type Session struct {
	ID      string `json:"id"`
	Date    string `json:"date"` // horário local, sem fuso: 2026-09-29T15:30:00
	Hybrid  bool   `json:"hybrid"`
	Expired bool   `json:"expired"`
}

func (c *Client) Sessions(ctx context.Context, movieID string, cityID int) ([]TheaterDay, error) {
	const pageSize = 200
	var all []TheaterDay
	for page := 1; page <= 20; page++ {
		var chunk []TheaterDay
		q := url.Values{
			"movieId":    {movieID},
			"cityId":     {strconv.Itoa(cityID)},
			"pageNumber": {strconv.Itoa(page)},
			"pageSize":   {strconv.Itoa(pageSize)},
		}
		if err := c.get(ctx, "/v1/sessions/movieAndCity", q, &chunk); err != nil {
			return nil, err
		}
		all = append(all, chunk...)
		if len(chunk) < pageSize {
			break
		}
	}
	return all, nil
}

// Seat é um elemento do mapa da sala.
type Seat struct {
	Row        int    `json:"row"`
	Col        int    `json:"col"`
	Name       string `json:"name"`
	Status     int    `json:"status"`
	Type       int    `json:"type"`
	Selectable bool   `json:"selectable"`
}

type SeatMap struct {
	Elements []Seat `json:"elements"`
}

func (c *Client) SeatMap(ctx context.Context, theaterID int, sessionID string) (*SeatMap, error) {
	var out SeatMap
	q := url.Values{"theaterId": {strconv.Itoa(theaterID)}, "sessionId": {sessionID}}
	if err := c.get(ctx, "/v1/seatmaps", q, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
