package kucoin

import (
	"context"
	"db_trace/kucoin-ising-bot/internal/config"
	"db_trace/kucoin-ising-bot/internal/core"
	"db_trace/kucoin-ising-bot/internal/types"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

type Client struct {
	cfg    config.Config
	symbol string

	events chan<- core.Event

	httpClient *http.Client
}

func NewClient(cfg config.Config, events chan<- core.Event) *Client {
	return &Client{
		cfg:        cfg,
		symbol:     cfg.Symbol,
		events:     events,
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

type bulletResponse struct {
	Code string `json:"code"`
	Data struct {
		Token           string `json:"token"`
		InstanceServers []struct {
			Endpoint     string `json:"endpoint"`
			Protocol     string `json:"protocol"`
			Encrypt      bool   `json:"encrypt"`
			PingInterval int64  `json:"pingInterval"`
			PingTimeout  int64  `json:"pingTimeout"`
		} `json:"instanceServers"`
	} `json:"data"`
}

type wsEnvelope struct {
	ID      string          `json:"id"`
	Type    string          `json:"type"`
	Topic   string          `json:"topic"`
	Subject string          `json:"subject"`
	Data    json.RawMessage `json:"data"`
}

type tickerData struct {
	Sequence    string `json:"sequence"`
	Price       string `json:"price"`
	Size        string `json:"size"`
	BestAsk     string `json:"bestAsk"`
	BestAskSize string `json:"bestAskSize"`
	BestBid     string `json:"bestBid"`
	BestBidSize string `json:"bestBidSize"`
}

type tradeData struct {
	Price    string `json:"price"`
	Sequence string `json:"sequence"`
	Side     string `json:"side"`
	Size     string `json:"size"`
	Symbol   string `json:"symbol"`
	Time     string `json:"time"`
}

func (c *Client) Run(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if err := c.runOnce(ctx); err != nil && !errors.Is(err, context.Canceled) {
			fmt.Printf("kucoin reconnect after error: %v\n", err)
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(c.cfg.ReconnectDelay):
		}
	}
}

func (c *Client) runOnce(ctx context.Context) error {
	wsURL, pingInterval, err := c.getWSURL(ctx)
	if err != nil {
		return fmt.Errorf("get ws url: %w", err)
	}

	conn, _, err := websocket.DefaultDialer.DialContext(ctx, wsURL, nil)
	if err != nil {
		return fmt.Errorf("dial ws: %w", err)
	}
	defer conn.Close()

	if err := c.waitWelcome(ctx, conn); err != nil {
		return fmt.Errorf("wait welcome: %w", err)
	}

	if err := c.subscribe(conn); err != nil {
		return fmt.Errorf("subscribe: %w", err)
	}

	if pingInterval <= 0 {
		pingInterval = 18_000
	}

	errCh := make(chan error, 2)

	go func() {
		ticker := time.NewTicker(time.Duration(pingInterval) * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				errCh <- ctx.Err()
				return
			case <-ticker.C:
				msg := map[string]any{
					"id":   strconv.FormatInt(time.Now().UnixNano(), 10),
					"type": "ping",
				}
				if err := conn.WriteJSON(msg); err != nil {
					errCh <- err
					return
				}
			}
		}
	}()

	go func() {
		for {
			select {
			case <-ctx.Done():
				errCh <- ctx.Err()
				return
			default:
			}

			_ = conn.SetReadDeadline(time.Now().Add(c.cfg.WSReadTimeout))

			_, payload, err := conn.ReadMessage()
			if err != nil {
				errCh <- err
				return
			}

			if err := c.handleMessage(payload); err != nil {
				errCh <- err
				return
			}
		}
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errCh:
		return err
	}
}

func (c *Client) waitWelcome(ctx context.Context, conn *websocket.Conn) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		_ = conn.SetReadDeadline(time.Now().Add(c.cfg.WSReadTimeout))

		_, payload, err := conn.ReadMessage()
		if err != nil {
			return err
		}

		var env wsEnvelope
		if err := json.Unmarshal(payload, &env); err != nil {
			continue
		}

		switch env.Type {
		case "welcome":
			return nil
		case "error":
			return fmt.Errorf("ws error before welcome: %s", string(payload))
		}
	}
}

func (c *Client) subscribe(conn *websocket.Conn) error {
	topics := []string{
		"/market/ticker:" + c.symbol,
		"/market/match:" + c.symbol,
	}

	for _, topic := range topics {
		msg := map[string]any{
			"id":       strconv.FormatInt(time.Now().UnixNano(), 10),
			"type":     "subscribe",
			"topic":    topic,
			"response": true,
		}
		if err := conn.WriteJSON(msg); err != nil {
			return err
		}
	}

	return nil
}

func (c *Client) handleMessage(payload []byte) error {
	var env wsEnvelope
	if err := json.Unmarshal(payload, &env); err != nil {
		return nil
	}

	switch env.Type {
	case "welcome", "ack", "pong":
		return nil
	case "error":
		return fmt.Errorf("ws error: %s", string(payload))
	case "message":
	default:
		return nil
	}

	switch {
	case strings.HasPrefix(env.Topic, "/market/ticker:"):
		return c.handleTicker(env)
	case strings.HasPrefix(env.Topic, "/market/match:"):
		return c.handleTrade(env)
	default:
		return nil
	}
}

func (c *Client) handleTicker(env wsEnvelope) error {
	var d tickerData
	if err := json.Unmarshal(env.Data, &d); err != nil {
		return nil
	}

	bestBid, err1 := strconv.ParseFloat(d.BestBid, 64)
	bestAsk, err2 := strconv.ParseFloat(d.BestAsk, 64)
	bidSize, err3 := strconv.ParseFloat(d.BestBidSize, 64)
	askSize, err4 := strconv.ParseFloat(d.BestAskSize, 64)
	if err1 != nil || err2 != nil || err3 != nil || err4 != nil {
		return nil
	}

	if bestBid <= 0 || bestAsk <= 0 || bestBid >= bestAsk || bidSize <= 0 || askSize <= 0 {
		return nil
	}

	book := types.BookSnapshot{
		Symbol: c.symbol,
		Bids: []types.Level{
			{Price: bestBid, Qty: bidSize},
		},
		Asks: []types.Level{
			{Price: bestAsk, Qty: askSize},
		},
		Timestamp: time.Now(),
	}

	c.emitBook(book)
	return nil
}

func (c *Client) handleTrade(env wsEnvelope) error {
	var d tradeData
	if err := json.Unmarshal(env.Data, &d); err != nil {
		return nil
	}

	price, err1 := strconv.ParseFloat(d.Price, 64)
	qty, err2 := strconv.ParseFloat(d.Size, 64)
	if err1 != nil || err2 != nil || price <= 0 || qty <= 0 {
		return nil
	}

	side := types.SideUnknown
	switch strings.ToLower(d.Side) {
	case "buy":
		side = types.SideBuy
	case "sell":
		side = types.SideSell
	}

	ts := parseKuCoinTime(d.Time)

	tr := types.TradeTick{
		Symbol:    c.symbol,
		Price:     price,
		Qty:       qty,
		Side:      side,
		Timestamp: ts,
	}

	c.emitTrade(tr)
	return nil
}

func (c *Client) emitBook(book types.BookSnapshot) {
	select {
	case c.events <- core.Event{Type: core.EventBook, Book: &book}:
	default:
	}
}

func (c *Client) emitTrade(tr types.TradeTick) {
	select {
	case c.events <- core.Event{Type: core.EventTrade, Trade: &tr}:
	default:
	}
}

func (c *Client) getWSURL(ctx context.Context) (string, int64, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.kucoin.com/api/v1/bullet-public", nil)
	if err != nil {
		return "", 0, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", 0, err
	}
	defer resp.Body.Close()

	var br bulletResponse
	if err := json.NewDecoder(resp.Body).Decode(&br); err != nil {
		return "", 0, err
	}
	if br.Code != "200000" {
		return "", 0, fmt.Errorf("unexpected code: %s", br.Code)
	}
	if len(br.Data.InstanceServers) == 0 {
		return "", 0, errors.New("no instanceServers")
	}

	srv := br.Data.InstanceServers[0]
	u, err := url.Parse(srv.Endpoint)
	if err != nil {
		return "", 0, err
	}

	q := u.Query()
	q.Set("token", br.Data.Token)
	q.Set("connectId", strconv.FormatInt(time.Now().UnixNano(), 10))
	u.RawQuery = q.Encode()

	return u.String(), srv.PingInterval, nil
}

func parseKuCoinTime(raw string) time.Time {
	if raw == "" {
		return time.Now()
	}

	n, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return time.Now()
	}

	switch {
	case n > 1e15:
		return time.Unix(0, n) // nanoseconds
	case n > 1e12:
		return time.UnixMilli(n) // milliseconds
	case n > 1e9:
		return time.Unix(n, 0) // seconds
	default:
		return time.Now()
	}
}
