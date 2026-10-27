package websocket

import (
	"encoding/json"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/MIZHT/inter-exchange-arbitrage/internal/core/domain"
	transportws "github.com/MIZHT/inter-exchange-arbitrage/internal/core/transport/websocket_manager"
	"github.com/MIZHT/inter-exchange-arbitrage/internal/exchange/mexc/websocket/mexc_proto"
	"google.golang.org/protobuf/proto"
)

type Manager = transportws.Router

func NewManager(proxy string, pools ...*transportws.ProxyPool) *Manager {
	pool := transportws.PoolOrProxy(proxy, pools...)
	var requestID atomic.Int64
	return transportws.NewRouter(transportws.Protocol{
		Name: "mexc",
		Endpoint: func(m transportws.Market) string {
			if m == transportws.Spot {
				return "wss://wbs-api.mexc.com/ws"
			}
			return "wss://contract.mexc.com/edge"
		},
		Capacity: func(m transportws.Market) int {
			if m == transportws.Spot {
				return 30
			}
			return 20
		},
		Compression: func(m transportws.Market) bool { return m == transportws.Spot },
		Subscribe: func(market transportws.Market, coins []string, add bool) []any {
			if market == transportws.Spot {
				method := "SUBSCRIPTION"
				if !add {
					method = "UNSUBSCRIPTION"
				}
				channels := make([]string, 0, len(coins))
				for _, coin := range coins {
					channels = append(channels, "spot@public.limit.depth.v3.api.pb@"+coin+"USDT@20")
				}
				return []any{map[string]any{"method": method, "params": channels, "id": requestID.Add(1)}}
			}
			messages := make([]any, 0, len(coins))
			method := "sub.depth.full"
			if !add {
				method = "usub.depth.full"
			}
			for _, coin := range coins {
				messages = append(messages, map[string]any{
					"method": method,
					"param":  map[string]any{"symbol": coin + "_USDT", "limit": 20},
					"id":     requestID.Add(1),
				})
			}
			return messages
		},
		Parse:        parseBook,
		PingInterval: 20 * time.Second,
		PingMessage: func(market transportws.Market) any {
			if market == transportws.Spot {
				return map[string]any{"method": "PING", "id": requestID.Add(1)}
			}
			return map[string]any{"method": "ping"}
		},
		Pong: handlePing,
	}, pool)
}

func handlePing(_ transportws.Market, raw []byte) ([]byte, bool) {
	text := strings.TrimSpace(string(raw))
	if text == "ping" {
		return []byte("pong"), true
	}
	var message struct {
		Method  string          `json:"method"`
		Channel string          `json:"channel"`
		Msg     string          `json:"msg"`
		ID      json.RawMessage `json:"id"`
	}
	if json.Unmarshal(raw, &message) != nil {
		return nil, false
	}
	if strings.EqualFold(message.Method, "ping") {
		response, _ := json.Marshal(map[string]any{"method": "pong", "id": message.ID})
		return response, true
	}
	if message.Msg == "ping" {
		response, _ := json.Marshal(map[string]any{"msg": "pong"})
		return response, true
	}
	if strings.EqualFold(message.Channel, "pong") || strings.EqualFold(message.Method, "pong") || message.Msg == "pong" {
		return nil, true
	}
	return nil, false
}

func parseBook(market transportws.Market, raw []byte) (transportws.Update, bool) {
	if market == transportws.Spot {
		var wrapper mexc_proto.PushDataV3ApiWrapper
		if proto.Unmarshal(raw, &wrapper) != nil {
			return transportws.Update{}, false
		}
		depth := wrapper.GetPublicLimitDepths()
		if depth == nil {
			return transportws.Update{}, false
		}
		coin := baseCoin(wrapper.GetSymbol())
		if coin == "" {
			coin = baseCoin(wrapper.GetSymbolId())
		}
		if coin == "" {
			return transportws.Update{}, false
		}
		book := domain.OrderBook{Market: coin}
		for _, level := range depth.GetBids() {
			if level == nil {
				continue
			}
			price, e1 := strconv.ParseFloat(level.GetPrice(), 64)
			quantity, e2 := strconv.ParseFloat(level.GetQuantity(), 64)
			if e1 == nil && e2 == nil && price > 0 && quantity > 0 {
				book.Bids = append(book.Bids, domain.BidAsk{Price: price, Quantity: quantity})
			}
		}
		for _, level := range depth.GetAsks() {
			if level == nil {
				continue
			}
			price, e1 := strconv.ParseFloat(level.GetPrice(), 64)
			quantity, e2 := strconv.ParseFloat(level.GetQuantity(), 64)
			if e1 == nil && e2 == nil && price > 0 && quantity > 0 {
				book.Asks = append(book.Asks, domain.BidAsk{Price: price, Quantity: quantity})
			}
		}
		return transportws.Update{Book: book, Snapshot: true}, true
	}

	var message struct {
		Channel string `json:"channel"`
		Symbol  string `json:"symbol"`
		Data    struct {
			Symbol string              `json:"symbol"`
			Bids   [][]json.RawMessage `json:"bids"`
			Asks   [][]json.RawMessage `json:"asks"`
		} `json:"data"`
	}
	if json.Unmarshal(raw, &message) != nil || message.Channel != "push.depth" {
		return transportws.Update{}, false
	}
	symbol := message.Symbol
	if symbol == "" {
		symbol = message.Data.Symbol
	}
	coin := baseCoin(symbol)
	if coin == "" {
		return transportws.Update{}, false
	}
	book := domain.OrderBook{Market: coin, Bids: parseLevels(message.Data.Bids), Asks: parseLevels(message.Data.Asks)}
	return transportws.Update{Book: book, Snapshot: true}, true
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
