package kucoin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/websocket"

	"db_trace/kucoin-ising-bot/internal/config"
	"db_trace/kucoin-ising-bot/internal/core"
	"db_trace/kucoin-ising-bot/internal/types"
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

// Futures level5
type depth5Data struct {
	Bids     [][]any `json:"bids"`
	Asks     [][]any `json:"asks"`
	Sequence int64   `json:"sequence"`
	Ts       int64   `json:"ts"`
}

// Futures ticker v1
type tickerV1Data struct {
	Symbol       string `json:"symbol"`
	Sequence     int64  `json:"sequence"`
	Side         string `json:"side"`
	Size         any    `json:"size"` // contracts
	Price        string `json:"price"`
	BestBidSize  any    `json:"bestBidSize"`
	BestBidPrice string `json:"bestBidPrice"`
	BestAskPrice string `json:"bestAskPrice"`
	BestAskSize  any    `json:"bestAskSize"`
	Ts           int64  `json:"ts"`
}

func (c *Client) Run(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if err := c.runOnce(ctx); err != nil && !errors.Is(err, context.Canceled) {
			fmt.Printf("kucoin futures reconnect after error: %v\n", err)
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
		return fmt.Errorf("get futures ws url: %w", err)
	}

	conn, _, err := websocket.DefaultDialer.DialContext(ctx, wsURL, nil)
	if err != nil {
		return fmt.Errorf("dial futures ws: %w", err)
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
		"/contractMarket/level2Depth5:" + c.symbol,
		"/contractMarket/ticker:" + c.symbol, // v1 => last match + side/size/price
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
	case strings.HasPrefix(env.Topic, "/contractMarket/level2Depth5:"):
		return c.handleDepth5(env)
	case strings.HasPrefix(env.Topic, "/contractMarket/ticker:"):
		return c.handleTickerV1(env)
	default:
		return nil
	}
}

func (c *Client) handleDepth5(env wsEnvelope) error {
	var d depth5Data
	if err := json.Unmarshal(env.Data, &d); err != nil {
		return nil
	}

	bids := make([]types.Level, 0, len(d.Bids))
	for _, row := range d.Bids {
		if len(row) < 2 {
			continue
		}
		price, ok1 := toFloat(row[0])
		sizeContracts, ok2 := toFloat(row[1])
		if !ok1 || !ok2 || price <= 0 || sizeContracts <= 0 {
			continue
		}

		// Переводим контракты в базовый актив, чтобы текущая математика paper-модели
		// оставалась совместимой с (exit-entry)*size.
		sizeBase := sizeContracts * c.cfg.ContractMultiplier
		bids = append(bids, types.Level{
			Price: price,
			Qty:   sizeBase,
		})
	}

	asks := make([]types.Level, 0, len(d.Asks))
	for _, row := range d.Asks {
		if len(row) < 2 {
			continue
		}
		price, ok1 := toFloat(row[0])
		sizeContracts, ok2 := toFloat(row[1])
		if !ok1 || !ok2 || price <= 0 || sizeContracts <= 0 {
			continue
		}

		sizeBase := sizeContracts * c.cfg.ContractMultiplier
		asks = append(asks, types.Level{
			Price: price,
			Qty:   sizeBase,
		})
	}

	if len(bids) == 0 || len(asks) == 0 {
		return nil
	}
	if bids[0].Price >= asks[0].Price {
		return nil
	}

	book := types.BookSnapshot{
		Symbol:    c.symbol,
		Bids:      bids,
		Asks:      asks,
		Timestamp: parseFuturesTS(d.Ts),
	}
	c.emitBook(book)
	return nil
}

func (c *Client) handleTickerV1(env wsEnvelope) error {
	var d tickerV1Data
	if err := json.Unmarshal(env.Data, &d); err != nil {
		return nil
	}

	price, err := strconv.ParseFloat(d.Price, 64)
	if err != nil || price <= 0 {
		return nil
	}

	sizeContracts, ok := anyToFloat(d.Size)
	if !ok || sizeContracts <= 0 {
		return nil
	}

	side := types.SideUnknown
	switch strings.ToLower(d.Side) {
	case "buy":
		side = types.SideBuy
	case "sell":
		side = types.SideSell
	}

	tr := types.TradeTick{
		Symbol:    c.symbol,
		Price:     price,
		Qty:       sizeContracts * c.cfg.ContractMultiplier,
		Side:      side,
		Timestamp: parseFuturesTS(d.Ts),
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
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		"https://api-futures.kucoin.com/api/v1/bullet-public",
		nil,
	)
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

func parseFuturesTS(ts int64) time.Time {
	switch {
	case ts > 1e15:
		return time.Unix(0, ts)
	case ts > 1e12:
		return time.UnixMilli(ts)
	case ts > 1e9:
		return time.Unix(ts, 0)
	default:
		return time.Now()
	}
}

func toFloat(v any) (float64, bool) {
	switch x := v.(type) {
	case float64:
		return x, true
	case string:
		f, err := strconv.ParseFloat(x, 64)
		return f, err == nil
	case json.Number:
		f, err := x.Float64()
		return f, err == nil
	case int:
		return float64(x), true
	case int64:
		return float64(x), true
	default:
		return 0, false
	}
}

func anyToFloat(v any) (float64, bool) {
	return toFloat(v)
}
