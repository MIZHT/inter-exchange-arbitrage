package websocket

import (
	"bytes"
	"compress/gzip"
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
		Name: "coinex",
		Endpoint: func(m transportws.Market) string {
			if m == transportws.Spot {
				return "wss://socket.coinex.com/v2/spot"
			}
			return "wss://socket.coinex.com/v2/futures"
		},
		Capacity: func(transportws.Market) int { return 20 },
		Decode:   decodeMessage,
		Subscribe: func(_ transportws.Market, coins []string, add bool) []any {
			method := "depth.subscribe"
			params := make(map[string]any)
			if add {
				markets := make([][]any, 0, len(coins))
				for _, coin := range coins {
					markets = append(markets, []any{coin + "USDT", 20, "0", true})
				}
				params["market_list"] = markets
			} else {
				method = "depth.unsubscribe"
				markets := make([]string, 0, len(coins))
				for _, coin := range coins {
					markets = append(markets, coin+"USDT")
				}
				params["market_list"] = markets
			}
			return []any{map[string]any{"method": method, "params": params, "id": requestID.Add(1)}}
		},
		Parse:        parseBook,
		PingInterval: 25 * time.Second,
		PingMessage: func(transportws.Market) any {
			return map[string]any{"method": "ping", "params": []any{}, "id": requestID.Add(1)}
		},
		Pong: handlePing,
	}, pool)
}

func decodeMessage(raw []byte) []byte {
	if json.Valid(raw) {
		return raw
	}
	reader, err := gzip.NewReader(bytes.NewReader(raw))
	if err != nil {
		return raw
	}
	defer reader.Close()
	decoded, err := io.ReadAll(reader)
	if err != nil {
		return raw
	}
	return decoded
}

func handlePing(_ transportws.Market, raw []byte) ([]byte, bool) {
	var message struct {
		Method string          `json:"method"`
		ID     json.RawMessage `json:"id"`
	}
	if json.Unmarshal(raw, &message) != nil {
		return nil, false
	}
	switch strings.ToLower(message.Method) {
	case "ping", "server.ping":
		response, _ := json.Marshal(map[string]any{"method": "server.pong", "params": []any{}, "id": message.ID})
		return response, true
	case "pong", "server.pong":
		return nil, true
	default:
		return nil, false
	}
}

func parseBook(_ transportws.Market, raw []byte) (transportws.Update, bool) {
	var message struct {
		Method string `json:"method"`
		Data   struct {
			Market string `json:"market"`
			IsFull bool   `json:"is_full"`
			Depth  struct {
				Bids [][]any `json:"bids"`
				Asks [][]any `json:"asks"`
			} `json:"depth"`
		} `json:"data"`
		Params []json.RawMessage `json:"params"`
	}
	if json.Unmarshal(raw, &message) != nil {
		return transportws.Update{}, false
	}
	if !strings.Contains(message.Method, "depth.update") {
		return transportws.Update{}, false
	}
	market, bids, asks, full := message.Data.Market, message.Data.Depth.Bids, message.Data.Depth.Asks, message.Data.IsFull
	if market == "" && len(message.Params) >= 2 {
		if len(message.Params) >= 3 {
			_ = json.Unmarshal(message.Params[2], &market)
		}
		_ = json.Unmarshal(message.Params[0], &full)
		var depth struct {
			Bids [][]any `json:"bids"`
			Asks [][]any `json:"asks"`
		}
		_ = json.Unmarshal(message.Params[1], &depth)
		bids, asks = depth.Bids, depth.Asks
	}
	coin := baseCoin(market)
	if coin == "" {
		return transportws.Update{}, false
	}
	book := domain.OrderBook{Market: coin, Bids: parseLevels(bids), Asks: parseLevels(asks)}
	return transportws.Update{Book: book, Snapshot: full}, true
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

func parseLevels(rows [][]any) []domain.BidAsk {
	levels := make([]domain.BidAsk, 0, len(rows))
	for _, row := range rows {
		if len(row) < 2 {
			continue
		}
		price, okPrice := number(row[0])
		quantity, okQuantity := number(row[1])
		if okPrice && okQuantity && price > 0 {
			levels = append(levels, domain.BidAsk{Price: price, Quantity: quantity})
		}
	}
	return levels
}

func number(value any) (float64, bool) {
	switch value := value.(type) {
	case string:
		n, err := strconv.ParseFloat(value, 64)
		return n, err == nil
	case float64:
		return value, true
	case json.Number:
		n, err := value.Float64()
		return n, err == nil
	default:
		return 0, false
	}
}
