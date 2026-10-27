package websocket

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"io"
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
		Name: "xt",
		Endpoint: func(m transportws.Market) string {
			if m == transportws.Spot {
				return "wss://stream.xt.com/public"
			}
			return "wss://fstream.xt.com/ws/market"
		},
		Capacity:    func(transportws.Market) int { return 20 },
		Compression: func(m transportws.Market) bool { return m == transportws.Futures },
		Decode:      decodeMessage,
		Subscribe: func(market transportws.Market, coins []string, add bool) []any {
			method, event := "subscribe", "depth@"
			if market == transportws.Futures {
				method, event = "SUBSCRIBE", "depth@"
			}
			if !add {
				method = "unsubscribe"
				if market == transportws.Futures {
					method = "UNSUBSCRIBE"
				}
			}
			params := make([]string, 0, len(coins))
			for _, coin := range coins {
				topic := event + strings.ToLower(coin+"_usdt") + ",20"
				if market == transportws.Futures {
					topic += ",1000ms"
				}
				params = append(params, topic)
			}
			return []any{map[string]any{"method": method, "params": params, "id": requestID.Add(1)}}
		},
		Parse:        parseBook,
		PingInterval: 20 * time.Second,
		PingText:     true,
		PingMessage:  func(transportws.Market) any { return "ping" },
		Pong:         handlePing,
	}, pool)
}

func decodeMessage(raw []byte) []byte {
	if json.Valid(raw) {
		return raw
	}
	candidates := [][]byte{raw}
	if decoded, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(raw))); err == nil {
		candidates = append(candidates, decoded)
	}
	if decoded, err := base64.RawStdEncoding.DecodeString(strings.TrimSpace(string(raw))); err == nil {
		candidates = append(candidates, decoded)
	}
	for _, candidate := range candidates {
		reader, err := gzip.NewReader(bytes.NewReader(candidate))
		if err != nil {
			if json.Valid(candidate) {
				return candidate
			}
			continue
		}
		decoded, readErr := io.ReadAll(reader)
		_ = reader.Close()
		if readErr == nil && json.Valid(decoded) {
			return decoded
		}
	}
	return raw
}

func handlePing(_ transportws.Market, raw []byte) ([]byte, bool) {
	text := strings.TrimSpace(string(raw))
	if text == "ping" {
		return []byte("pong"), true
	}
	if text == "pong" {
		return nil, true
	}
	var message map[string]json.RawMessage
	if json.Unmarshal(raw, &message) != nil {
		return nil, false
	}
	if timestamp, ok := message["ping"]; ok {
		response, _ := json.Marshal(map[string]json.RawMessage{"pong": timestamp})
		return response, true
	}
	var method string
	if json.Unmarshal(message["method"], &method) == nil {
		switch strings.ToLower(method) {
		case "ping":
			return []byte(`{"method":"pong"}`), true
		case "pong":
			return nil, true
		}
	}
	return nil, false
}

func parseBook(_ transportws.Market, raw []byte) (transportws.Update, bool) {
	var message struct {
		Method  string            `json:"method"`
		Topic   string            `json:"topic"`
		Event   string            `json:"event"`
		Channel string            `json:"channel"`
		Data    json.RawMessage   `json:"data"`
		Params  []json.RawMessage `json:"params"`
	}
	if json.Unmarshal(raw, &message) != nil {
		return transportws.Update{}, false
	}
	data := message.Data
	if len(data) == 0 && len(message.Params) > 0 {
		data = message.Params[0]
	}
	var payload struct {
		Market   string          `json:"market"`
		Symbol   string          `json:"symbol"`
		Contract string          `json:"contract"`
		Channel  string          `json:"channel"`
		IsFull   *bool           `json:"isFull"`
		IsFull2  *bool           `json:"is_full"`
		Bids     json.RawMessage `json:"bids"`
		Asks     json.RawMessage `json:"asks"`
		BidRows  json.RawMessage `json:"b"`
		AskRows  json.RawMessage `json:"a"`
	}
	if json.Unmarshal(data, &payload) != nil {
		var items []json.RawMessage
		if json.Unmarshal(data, &items) != nil || len(items) == 0 || json.Unmarshal(items[0], &payload) != nil {
			return transportws.Update{}, false
		}
	}
	channel := strings.ToLower(message.Channel)
	if payload.Channel != "" {
		channel = strings.ToLower(payload.Channel)
	}
	if !strings.Contains(strings.ToLower(message.Method), "depth") &&
		!strings.Contains(strings.ToLower(message.Topic), "depth") &&
		!strings.Contains(strings.ToLower(message.Event), "depth") &&
		!strings.Contains(channel, "depth") && payload.Market == "" && payload.Symbol == "" && payload.Contract == "" {
		return transportws.Update{}, false
	}
	symbol := payload.Symbol
	if symbol == "" {
		symbol = payload.Market
	}
	if symbol == "" {
		symbol = payload.Contract
	}
	if symbol == "" {
		symbol = symbolFromTopic(message.Event)
	}
	if symbol == "" {
		symbol = symbolFromTopic(message.Topic)
	}
	coin := baseCoin(symbol)
	if coin == "" {
		return transportws.Update{}, false
	}
	bidRows, askRows := payload.Bids, payload.Asks
	if len(bidRows) == 0 {
		bidRows = payload.BidRows
	}
	if len(askRows) == 0 {
		askRows = payload.AskRows
	}
	bids, err := parseLevels(bidRows)
	if err != nil {
		return transportws.Update{}, false
	}
	asks, err := parseLevels(askRows)
	if err != nil {
		return transportws.Update{}, false
	}
	snapshot := true
	if payload.IsFull != nil {
		snapshot = *payload.IsFull
	} else if payload.IsFull2 != nil {
		snapshot = *payload.IsFull2
	} else if strings.Contains(strings.ToLower(message.Event+" "+message.Topic), "depth_update") {
		snapshot = false
	}
	book := domain.OrderBook{Market: coin, Bids: bids, Asks: asks}
	return transportws.Update{Book: book, Snapshot: snapshot}, true
}

func symbolFromTopic(topic string) string {
	for _, prefix := range []string{"depth_update@", "depth@"} {
		if strings.HasPrefix(strings.ToLower(topic), prefix) {
			value := topic[len(prefix):]
			if i := strings.IndexByte(value, ','); i >= 0 {
				value = value[:i]
			}
			return value
		}
	}
	return ""
}

func baseCoin(symbol string) string {
	symbol = strings.ToUpper(strings.TrimSpace(symbol))
	symbol = strings.ReplaceAll(symbol, "/", "_")
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
			price, ok1 := number(pair[0])
			quantity, ok2 := number(pair[1])
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
		quantity, ok := number(level.Quantity)
		if err == nil && ok && price > 0 {
			levels = append(levels, domain.BidAsk{Price: price, Quantity: quantity})
		}
	}
	return levels, nil
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
