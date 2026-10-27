package websocket

import (
	"encoding/json"
	"strconv"
	"strings"
	"sync/atomic"

	"github.com/MIZHT/inter-exchange-arbitrage/internal/core/domain"
	transportws "github.com/MIZHT/inter-exchange-arbitrage/internal/core/transport/websocket_manager"
)

type Manager = transportws.Router

func NewManager(proxy string, pools ...*transportws.ProxyPool) *Manager {
	pool := transportws.PoolOrProxy(proxy, pools...)
	var requestID atomic.Int64
	return transportws.NewRouter(transportws.Protocol{
		Name: "binance",
		Endpoint: func(m transportws.Market) string {
			if m == transportws.Spot {
				return "wss://stream.binance.com:9443/ws"
			}
			return "wss://fstream.binance.com/ws"
		},
		Capacity: func(transportws.Market) int { return 25 },
		Subscribe: func(_ transportws.Market, coins []string, add bool) []any {
			params := make([]string, 0, len(coins))
			for _, coin := range coins {
				params = append(params, strings.ToLower(coin)+"usdt@depth20@1000ms")
			}
			method := "SUBSCRIBE"
			if !add {
				method = "UNSUBSCRIBE"
			}
			return []any{map[string]any{"method": method, "params": params, "id": requestID.Add(1)}}
		},
		Parse: parseBook,
	}, pool)
}

func parseBook(_ transportws.Market, raw []byte) (transportws.Update, bool) {
	var wrapped struct {
		Data *struct {
			Symbol       string     `json:"s"`
			EventTime    int64      `json:"E"`
			LastUpdateID int64      `json:"lastUpdateId"`
			Bids         [][]string `json:"bids"`
			Asks         [][]string `json:"asks"`
		} `json:"data"`
	}
	_ = json.Unmarshal(raw, &wrapped)
	payload := wrapped.Data
	if payload == nil {
		var direct struct {
			Symbol       string     `json:"s"`
			EventTime    int64      `json:"E"`
			LastUpdateID int64      `json:"lastUpdateId"`
			Bids         [][]string `json:"bids"`
			Asks         [][]string `json:"asks"`
		}
		if json.Unmarshal(raw, &direct) != nil {
			return transportws.Update{}, false
		}
		payload = &direct
	}
	coin := strings.TrimSuffix(strings.ToUpper(payload.Symbol), "USDT")
	if coin == "" || coin == strings.ToUpper(payload.Symbol) {
		return transportws.Update{}, false
	}
	book := domain.OrderBook{Market: coin, Bids: levels(payload.Bids), Asks: levels(payload.Asks)}
	return transportws.Update{Book: book, Snapshot: true}, payload.LastUpdateID > 0 || len(book.Bids)+len(book.Asks) > 0
}
func levels(rows [][]string) []domain.BidAsk {
	out := make([]domain.BidAsk, 0, len(rows))
	for _, row := range rows {
		if len(row) < 2 {
			continue
		}
		p, e1 := strconv.ParseFloat(row[0], 64)
		q, e2 := strconv.ParseFloat(row[1], 64)
		if e1 == nil && e2 == nil && p > 0 {
			out = append(out, domain.BidAsk{Price: p, Quantity: q})
		}
	}
	return out
}
