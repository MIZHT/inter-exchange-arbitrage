package websocket

import (
	"encoding/json"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/MIZHT/inter-exchange-arbitrage/internal/core/domain"
	transportws "github.com/MIZHT/inter-exchange-arbitrage/internal/core/transport/websocket_manager"
)

type Manager = transportws.Router

func NewManager(proxy string, pools ...*transportws.ProxyPool) *Manager {
	pool := transportws.PoolOrProxy(proxy, pools...)
	var requestID atomic.Int64
	return transportws.NewRouter(transportws.Protocol{
		Name: "okx",
		Endpoint: func(transportws.Market) string {
			return "wss://ws.okx.com/ws/v5/public"
		},
		Capacity: func(transportws.Market) int { return 20 },
		Subscribe: func(market transportws.Market, coins []string, add bool) []any {
			op := "subscribe"
			if !add {
				op = "unsubscribe"
			}
			args := make([]map[string]string, 0, len(coins))
			for _, coin := range coins {
				instrument := coin + "-USDT"
				if market == transportws.Futures {
					instrument += "-SWAP"
				}
				args = append(args, map[string]string{"channel": "books5", "instId": instrument})
			}
			return []any{map[string]any{"id": strconv.FormatInt(requestID.Add(1), 10), "op": op, "args": args}}
		},
		Parse:        parseBook,
		PingInterval: 20 * time.Second,
		PingText:     true,
		PingMessage:  func(transportws.Market) any { return "ping" },
		Pong: func(_ transportws.Market, raw []byte) ([]byte, bool) {
			text := strings.TrimSpace(string(raw))
			return nil, text == "pong"
		},
	}, pool)
}

func parseBook(_ transportws.Market, raw []byte) (transportws.Update, bool) {
	var message struct {
		Arg struct {
			Channel string `json:"channel"`
			InstID  string `json:"instId"`
		} `json:"arg"`
		Data []struct {
			InstID string              `json:"instId"`
			Bids   [][]json.RawMessage `json:"bids"`
			Asks   [][]json.RawMessage `json:"asks"`
		} `json:"data"`
	}
	if json.Unmarshal(raw, &message) != nil || message.Arg.Channel != "books5" || len(message.Data) == 0 {
		return transportws.Update{}, false
	}
	item := message.Data[0]
	symbol := item.InstID
	if symbol == "" {
		symbol = message.Arg.InstID
	}
	coin := baseCoin(symbol)
	if coin == "" {
		return transportws.Update{}, false
	}
	book := domain.OrderBook{Market: coin, Bids: parseLevels(item.Bids), Asks: parseLevels(item.Asks)}
	return transportws.Update{Book: book, Snapshot: true}, len(book.Bids)+len(book.Asks) > 0
}

func baseCoin(symbol string) string {
	symbol = strings.ToUpper(strings.TrimSpace(symbol))
	for _, suffix := range []string{"-USDT-SWAP", "-USDT"} {
		if strings.HasSuffix(symbol, suffix) {
			return strings.TrimSuffix(symbol, suffix)
		}
	}
	return ""
}

func parseLevels(rows [][]json.RawMessage) []domain.BidAsk {
	levels := make([]domain.BidAsk, 0, len(rows))
	for _, row := range rows {
		if len(row) < 2 {
			continue
		}
		price, ok1 := number(row[0])
		quantity, ok2 := number(row[1])
		if ok1 && ok2 && price > 0 && quantity > 0 {
			levels = append(levels, domain.BidAsk{Price: price, Quantity: quantity})
		}
	}
	return levels
}

func number(raw json.RawMessage) (float64, bool) {
	var text string
	if json.Unmarshal(raw, &text) == nil {
		value, err := strconv.ParseFloat(text, 64)
		return value, err == nil
	}
	var value float64
	if err := json.Unmarshal(raw, &value); err != nil {
		return 0, false
	}
	return value, true
}
