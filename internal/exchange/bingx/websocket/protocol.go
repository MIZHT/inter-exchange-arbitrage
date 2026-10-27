package websocket

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync/atomic"

	"github.com/MIZHT/inter-exchange-arbitrage/internal/core/domain"
	transportws "github.com/MIZHT/inter-exchange-arbitrage/internal/core/transport/websocket_manager"
)

type Manager = transportws.Router

func NewManager(proxy string, pools ...*transportws.ProxyPool) *Manager {
	pool := transportws.PoolOrProxy(proxy, pools...)
	var seq atomic.Int64
	return transportws.NewRouter(transportws.Protocol{
		Name: "bingx", Endpoint: func(m transportws.Market) string {
			if m == transportws.Spot {
				return "wss://open-api-ws.bingx.com/market"
			}
			return "wss://open-api-swap.bingx.com/swap-market"
		}, Capacity: func(transportws.Market) int { return 25 },
		Subscribe: func(_ transportws.Market, coins []string, add bool) []any {
			req := "sub"
			if !add {
				req = "unsub"
			}
			result := make([]any, 0, len(coins))
			for _, coin := range coins {
				interval := "500ms"
				if coin == "BTC" || coin == "ETH" {
					interval = "200ms"
				}
				result = append(result, map[string]any{"id": fmt.Sprint(seq.Add(1)), "reqType": req, "dataType": coin + "-USDT@depth20@" + interval})
			}
			return result
		},
		Parse: parse,
		Pong: func(_ transportws.Market, raw []byte) ([]byte, bool) {
			if decoded, err := gunzip(raw); err == nil {
				raw = decoded
			}
			if strings.TrimSpace(string(raw)) == "ping" {
				return []byte("pong"), true
			}
			return nil, false
		},
	}, pool)
}

func parse(_ transportws.Market, raw []byte) (transportws.Update, bool) {
	if decoded, e := gunzip(raw); e == nil {
		raw = decoded
	} else {
		return transportws.Update{}, false
	}
	var msg struct {
		DataType string `json:"dataType"`
		Data     struct {
			Bids [][]string `json:"bids"`
			Asks [][]string `json:"asks"`
		} `json:"data"`
	}
	if json.Unmarshal(raw, &msg) != nil || !strings.Contains(msg.DataType, "depth") {
		return transportws.Update{}, false
	}
	symbol := strings.SplitN(msg.DataType, "@", 2)[0]
	coin := strings.TrimSuffix(strings.ReplaceAll(strings.ToUpper(symbol), "-", ""), "USDT")
	if coin == "" {
		return transportws.Update{}, false
	}
	book := domain.OrderBook{Market: coin, Bids: levels(msg.Data.Bids), Asks: levels(msg.Data.Asks)}
	return transportws.Update{Book: book, Snapshot: true}, true
}

func gunzip(raw []byte) ([]byte, error) {
	if len(raw) < 2 || raw[0] != 0x1f || raw[1] != 0x8b {
		return raw, nil
	}
	z, e := gzip.NewReader(bytes.NewReader(raw))
	if e != nil {
		return nil, e
	}
	defer z.Close()
	return io.ReadAll(z)
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
