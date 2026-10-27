package websocket

import (
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"github.com/MIZHT/inter-exchange-arbitrage/internal/core/domain"
	transportws "github.com/MIZHT/inter-exchange-arbitrage/internal/core/transport/websocket_manager"
)

type Manager = transportws.Router

func NewManager(proxy string, pools ...*transportws.ProxyPool) *Manager {
	pool := transportws.PoolOrProxy(proxy, pools...)
	return transportws.NewRouter(transportws.Protocol{
		Name: "bybit", Endpoint: func(m transportws.Market) string {
			if m == transportws.Spot {
				return "wss://stream.bybit.com/v5/public/spot"
			}
			return "wss://stream.bybit.com/v5/public/linear"
		}, Capacity: func(transportws.Market) int { return 10 },
		Subscribe: func(_ transportws.Market, coins []string, add bool) []any {
			args := make([]string, 0, len(coins))
			for _, coin := range coins {
				args = append(args, "orderbook.50."+coin+"USDT")
			}
			op := "subscribe"
			if !add {
				op = "unsubscribe"
			}
			return []any{map[string]any{"op": op, "args": args}}
		},
		Parse: parse, PingInterval: 20 * time.Second, PingMessage: func(transportws.Market) any { return map[string]any{"op": "ping"} },
	}, pool)
}

func parse(_ transportws.Market, raw []byte) (transportws.Update, bool) {
	var msg struct {
		Topic string `json:"topic"`
		Type  string `json:"type"`
		Data  struct {
			Symbol string              `json:"s"`
			Bids   [][]json.RawMessage `json:"b"`
			Asks   [][]json.RawMessage `json:"a"`
		} `json:"data"`
	}
	if json.Unmarshal(raw, &msg) != nil || !strings.Contains(msg.Topic, "orderbook") {
		return transportws.Update{}, false
	}
	symbol := msg.Data.Symbol
	if symbol == "" {
		parts := strings.Split(msg.Topic, ".")
		symbol = parts[len(parts)-1]
	}
	coin := strings.TrimSuffix(strings.ToUpper(symbol), "USDT")
	if coin == symbol || coin == "" {
		return transportws.Update{}, false
	}
	book := domain.OrderBook{Market: coin, Bids: levels(msg.Data.Bids), Asks: levels(msg.Data.Asks)}
	return transportws.Update{Book: book, Snapshot: msg.Type == "snapshot"}, true
}
func levels(rows [][]json.RawMessage) []domain.BidAsk {
	out := make([]domain.BidAsk, 0, len(rows))
	for _, row := range rows {
		if len(row) < 2 {
			continue
		}
		p, ok1 := number(row[0])
		q, ok2 := number(row[1])
		if ok1 && ok2 && p > 0 {
			out = append(out, domain.BidAsk{Price: p, Quantity: q})
		}
	}
	return out
}
func number(raw json.RawMessage) (float64, bool) {
	var text string
	if json.Unmarshal(raw, &text) == nil {
		v, e := strconv.ParseFloat(text, 64)
		return v, e == nil
	}
	var v float64
	e := json.Unmarshal(raw, &v)
	return v, e == nil
}
