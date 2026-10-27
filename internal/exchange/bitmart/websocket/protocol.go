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
		Name: "bitmart", Endpoint: func(m transportws.Market) string {
			if m == transportws.Spot {
				return "wss://ws-manager-compress.bitmart.com/api?protocol=1.1"
			}
			return "wss://openapi-ws-v2.bitmart.com/api?protocol=1.1"
		}, Capacity: func(transportws.Market) int { return 20 },
		Subscribe: func(m transportws.Market, coins []string, add bool) []any {
			args := make([]string, 0, len(coins))
			for _, coin := range coins {
				if m == transportws.Spot {
					args = append(args, "spot/depth20:"+coin+"_USDT")
				} else {
					args = append(args, "futures/depthAll20:"+coin+"USDT@200ms")
				}
			}
			action := "subscribe"
			if !add {
				action = "unsubscribe"
			}
			return []any{map[string]any{"action": action, "args": args}}
		},
		Parse: parse, PingInterval: 10 * time.Second, PingMessage: func(transportws.Market) any { return "ping" }, PingText: true, Pong: func(_ transportws.Market, raw []byte) ([]byte, bool) {
			text := strings.TrimSpace(string(raw))
			if text == "ping" {
				return []byte("pong"), true
			}
			return nil, text == "pong"
		},
	}, pool)
}

func parse(market transportws.Market, raw []byte) (transportws.Update, bool) {
	var msg map[string]json.RawMessage
	if json.Unmarshal(raw, &msg) != nil {
		return transportws.Update{}, false
	}
	var action, table, group string
	_ = json.Unmarshal(msg["action"], &action)
	_ = json.Unmarshal(msg["table"], &table)
	_ = json.Unmarshal(msg["group"], &group)
	value, marker := table, "depth"
	if market == transportws.Futures {
		value, marker = group, "futures/depthAll"
	}
	if !strings.Contains(value, marker) {
		return transportws.Update{}, false
	}
	var payload map[string]json.RawMessage
	if market == transportws.Spot {
		var rows []map[string]json.RawMessage
		if json.Unmarshal(msg["data"], &rows) != nil || len(rows) == 0 {
			return transportws.Update{}, false
		}
		payload = rows[0]
	} else {
		if json.Unmarshal(msg["data"], &payload) != nil {
			var rows []map[string]json.RawMessage
			if json.Unmarshal(msg["data"], &rows) != nil || len(rows) == 0 {
				return transportws.Update{}, false
			}
			payload = rows[0]
		}
		var nested map[string]json.RawMessage
		if json.Unmarshal(payload["depth"], &nested) == nil {
			payload = nested
		}
	}
	var symbol string
	_ = json.Unmarshal(payload["symbol"], &symbol)
	if symbol == "" {
		_ = json.Unmarshal(payload["s"], &symbol)
	}
	coin := strings.TrimSuffix(strings.TrimSuffix(strings.ToUpper(symbol), "_USDT"), "USDT")
	if coin == "" {
		return transportws.Update{}, false
	}
	bids := parseLevels(payload["bids"])
	if len(bids) == 0 {
		bids = parseLevels(payload["b"])
	}
	asks := parseLevels(payload["asks"])
	if len(asks) == 0 {
		asks = parseLevels(payload["a"])
	}
	return transportws.Update{Book: domain.OrderBook{Market: coin, Bids: bids, Asks: asks}, Snapshot: market == transportws.Futures || action != "update"}, true
}

func parseLevels(raw json.RawMessage) []domain.BidAsk {
	var rows []json.RawMessage
	if json.Unmarshal(raw, &rows) != nil {
		return nil
	}
	out := make([]domain.BidAsk, 0, len(rows))
	for _, row := range rows {
		var tuple []json.RawMessage
		if json.Unmarshal(row, &tuple) == nil && len(tuple) >= 2 {
			p, ok1 := number(tuple[0])
			q, ok2 := number(tuple[1])
			if ok1 && ok2 && p > 0 {
				out = append(out, domain.BidAsk{Price: p, Quantity: q})
			}
			continue
		}
		var obj map[string]json.RawMessage
		if json.Unmarshal(row, &obj) != nil {
			continue
		}
		p, ok1 := first(obj, "price", "p", "px")
		q, ok2 := first(obj, "size", "s", "qty", "quantity", "vol", "v", "amount")
		if ok1 && ok2 && p > 0 {
			out = append(out, domain.BidAsk{Price: p, Quantity: q})
		}
	}
	return out
}
func first(m map[string]json.RawMessage, keys ...string) (float64, bool) {
	for _, key := range keys {
		if raw, ok := m[key]; ok {
			return number(raw)
		}
	}
	return 0, false
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
