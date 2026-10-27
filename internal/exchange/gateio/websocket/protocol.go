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
		Name: "gateio",
		Endpoint: func(m transportws.Market) string {
			if m == transportws.Spot {
				return "wss://api.gateio.ws/ws/v4/"
			}
			return "wss://fx-ws.gateio.ws/v4/ws/usdt"
		},
		Capacity: func(transportws.Market) int { return 20 },
		Subscribe: func(market transportws.Market, coins []string, add bool) []any {
			channel := "spot.order_book"
			if market == transportws.Futures {
				channel = "futures.order_book"
			}
			event := "subscribe"
			if !add {
				event = "unsubscribe"
			}
			messages := make([]any, 0, len(coins))
			for _, coin := range coins {
				payload := []string{coin + "_USDT", "5", "100ms"}
				if market == transportws.Futures {
					payload[2] = "0"
				}
				messages = append(messages, map[string]any{
					"time":    time.Now().Unix(),
					"channel": channel,
					"event":   event,
					"payload": payload,
					"id":      requestID.Add(1),
				})
			}
			return messages
		},
		Parse:        parseBook,
		PingInterval: 20 * time.Second,
		PingMessage: func(market transportws.Market) any {
			channel := "spot.ping"
			if market == transportws.Futures {
				channel = "futures.ping"
			}
			return map[string]any{"time": time.Now().Unix(), "channel": channel}
		},
	}, pool)
}

func parseBook(market transportws.Market, raw []byte) (transportws.Update, bool) {
	var message struct {
		Channel string          `json:"channel"`
		Event   string          `json:"event"`
		Result  json.RawMessage `json:"result"`
	}
	if json.Unmarshal(raw, &message) != nil || message.Event == "subscribe" || message.Event == "unsubscribe" {
		return transportws.Update{}, false
	}
	wantedChannel := "spot.order_book"
	if market == transportws.Futures {
		wantedChannel = "futures.order_book"
	}
	if message.Channel != wantedChannel || len(message.Result) == 0 || string(message.Result) == "null" {
		return transportws.Update{}, false
	}
	var result struct {
		Symbol   string          `json:"s"`
		Contract string          `json:"contract"`
		Bids     json.RawMessage `json:"bids"`
		Asks     json.RawMessage `json:"asks"`
		BidRows  json.RawMessage `json:"b"`
		AskRows  json.RawMessage `json:"a"`
	}
	if json.Unmarshal(message.Result, &result) != nil {
		return transportws.Update{}, false
	}
	symbol := result.Symbol
	if symbol == "" {
		symbol = result.Contract
	}
	coin := baseCoin(symbol)
	if coin == "" {
		return transportws.Update{}, false
	}
	bidRows, askRows := result.Bids, result.Asks
	if len(bidRows) == 0 {
		bidRows = result.BidRows
	}
	if len(askRows) == 0 {
		askRows = result.AskRows
	}
	bids, err := parseLevels(bidRows)
	if err != nil {
		return transportws.Update{}, false
	}
	ask, err := parseLevels(askRows)
	if err != nil {
		return transportws.Update{}, false
	}
	snapshot := market == transportws.Spot || message.Event == "all"
	return transportws.Update{Book: domain.OrderBook{Market: coin, Bids: bids, Asks: ask}, Snapshot: snapshot}, true
}

func baseCoin(symbol string) string {
	symbol = strings.ToUpper(strings.TrimSpace(symbol))
	for _, suffix := range []string{"_USDT", "-USDT", "USDT"} {
		if strings.HasSuffix(symbol, suffix) {
			return strings.TrimSuffix(symbol, suffix)
		}
	}
	return ""
}

func parseLevels(raw json.RawMessage) ([]domain.BidAsk, error) {
	var rows []json.RawMessage
	if err := json.Unmarshal(raw, &rows); err != nil {
		return nil, err
	}
	levels := make([]domain.BidAsk, 0, len(rows))
	for _, row := range rows {
		var pair []json.RawMessage
		if json.Unmarshal(row, &pair) == nil && len(pair) >= 2 {
			price, ok1 := parseNumber(pair[0])
			quantity, ok2 := parseNumber(pair[1])
			if ok1 && ok2 && price > 0 {
				levels = append(levels, domain.BidAsk{Price: price, Quantity: quantity})
			}
			continue
		}
		var level struct {
			Price    string          `json:"p"`
			Quantity json.RawMessage `json:"s"`
		}
		if json.Unmarshal(row, &level) != nil {
			continue
		}
		price, err := strconv.ParseFloat(level.Price, 64)
		quantity, ok := parseNumber(level.Quantity)
		if err == nil && ok && price > 0 {
			levels = append(levels, domain.BidAsk{Price: price, Quantity: quantity})
		}
	}
	return levels, nil
}

func parseNumber(raw json.RawMessage) (float64, bool) {
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
