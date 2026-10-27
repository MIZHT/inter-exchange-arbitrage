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
		Name: "bitget", Endpoint: func(transportws.Market) string { return "wss://ws.bitget.com/v2/ws/public" }, Capacity: func(transportws.Market) int { return 40 },
		Subscribe: func(m transportws.Market, coins []string, add bool) []any {
			instType, channel := "SPOT", "books15"
			if m == transportws.Futures {
				instType, channel = "USDT-FUTURES", "books5"
			}
			args := make([]map[string]string, 0, len(coins))
			for _, coin := range coins {
				args = append(args, map[string]string{"instType": instType, "channel": channel, "instId": coin + "USDT"})
			}
			op := "subscribe"
			if !add {
				op = "unsubscribe"
			}
			return []any{map[string]any{"op": op, "args": args}}
		},
		Parse: parse, PingInterval: 25 * time.Second, PingMessage: func(transportws.Market) any { return "ping" }, PingText: true, Pong: func(_ transportws.Market, raw []byte) ([]byte, bool) {
			text := strings.TrimSpace(string(raw))
			if text == "ping" {
				return []byte("pong"), true
			}
			return nil, text == "pong"
		},
	}, pool)
}

func parse(_ transportws.Market, raw []byte) (transportws.Update, bool) {
	var msg struct {
		Action string `json:"action"`
		Arg    struct {
			InstID  string `json:"instId"`
			Channel string `json:"channel"`
		} `json:"arg"`
		Data []struct {
			InstID string     `json:"instId"`
			Bids   [][]string `json:"bids"`
			Asks   [][]string `json:"asks"`
		} `json:"data"`
	}
	if json.Unmarshal(raw, &msg) != nil || len(msg.Data) == 0 {
		return transportws.Update{}, false
	}
	d := msg.Data[0]
	symbol := d.InstID
	if symbol == "" {
		symbol = msg.Arg.InstID
	}
	coin := strings.TrimSuffix(strings.ToUpper(symbol), "USDT")
	if coin == "" || coin == strings.ToUpper(symbol) {
		return transportws.Update{}, false
	}
	book := domain.OrderBook{Market: coin, Bids: levels(d.Bids), Asks: levels(d.Asks)}
	return transportws.Update{Book: book, Snapshot: msg.Action == "snapshot" || strings.HasPrefix(msg.Arg.Channel, "books")}, true
}
func levels(rows [][]string) []domain.BidAsk {
	out := make([]domain.BidAsk, 0, len(rows))
	for _, r := range rows {
		if len(r) < 2 {
			continue
		}
		p, e1 := strconv.ParseFloat(r[0], 64)
		q, e2 := strconv.ParseFloat(r[1], 64)
		if e1 == nil && e2 == nil && p > 0 {
			out = append(out, domain.BidAsk{Price: p, Quantity: q})
		}
	}
	return out
}
