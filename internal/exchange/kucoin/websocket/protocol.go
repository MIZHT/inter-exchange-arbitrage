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
		Name: "kucoin",
		Endpoint: func(m transportws.Market) string {
			if m == transportws.Spot {
				return "wss://x-push-spot.kucoin.com"
			}
			return "wss://x-push-futures.kucoin.com"
		},
		Capacity: func(transportws.Market) int { return 20 },
		Subscribe: func(market transportws.Market, coins []string, add bool) []any {
			action, tradeType := "SUBSCRIBE", "SPOT"
			if !add {
				action = "UNSUBSCRIBE"
			}
			if market == transportws.Futures {
				tradeType = "FUTURES"
			}
			messages := make([]any, 0, len(coins))
			for _, coin := range coins {
				symbol := coin + "-USDT"
				if market == transportws.Futures {
					symbol = coin + "USDTM"
					if coin == "BTC" {
						symbol = "XBTUSDTM"
					}
				}
				messages = append(messages, map[string]any{
					"id":        requestID.Add(1),
					"action":    action,
					"channel":   "obu",
					"tradeType": tradeType,
					"symbol":    symbol,
					"depth":     "5",
				})
			}
			return messages
		},
		Parse:        parseBook,
		PingInterval: 20 * time.Second,
		PingMessage: func(transportws.Market) any {
			return map[string]any{"id": requestID.Add(1), "type": "ping"}
		},
		Pong: handlePing,
	}, pool)
}

func handlePing(_ transportws.Market, raw []byte) ([]byte, bool) {
	var message struct {
		ID   json.RawMessage `json:"id"`
		Type string          `json:"type"`
	}
	if json.Unmarshal(raw, &message) != nil || message.Type == "" {
		return nil, false
	}
	switch strings.ToLower(message.Type) {
	case "ping":
		response, _ := json.Marshal(map[string]any{"id": message.ID, "type": "pong"})
		return response, true
	case "pong":
		return nil, true
	default:
		return nil, false
	}
}

func parseBook(_ transportws.Market, raw []byte) (transportws.Update, bool) {
	var message struct {
		Topic string `json:"T"`
		Depth string `json:"dp"`
		Type  string `json:"t"`
		Data  struct {
			Symbol string              `json:"s"`
			Bids   [][]json.RawMessage `json:"b"`
			Asks   [][]json.RawMessage `json:"a"`
		} `json:"d"`
	}
	if json.Unmarshal(raw, &message) != nil || !strings.HasPrefix(strings.ToLower(message.Topic), "obu.") {
		return transportws.Update{}, false
	}
	coin := baseCoin(message.Data.Symbol)
	if coin == "" {
		return transportws.Update{}, false
	}
	bids, asks := parseLevels(message.Data.Bids), parseLevels(message.Data.Asks)
	if len(bids)+len(asks) == 0 {
		return transportws.Update{}, false
	}
	// The depth=5 subscription produces complete snapshots; incremental modes
	// remain supported if the subscription depth is changed later.
	snapshot := !strings.EqualFold(message.Type, "delta") && !strings.HasPrefix(message.Depth, "increment")
	return transportws.Update{Book: domain.OrderBook{Market: coin, Bids: bids, Asks: asks}, Snapshot: snapshot}, true
}

func baseCoin(symbol string) string {
	symbol = strings.ToUpper(strings.TrimSpace(symbol))
	if symbol == "XBTUSDTM" || symbol == "XBT-USDT" {
		return "BTC"
	}
	for _, suffix := range []string{"-USDT", "USDTM", "USDT"} {
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
		if ok1 && ok2 && price > 0 {
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
