package cmc

import (
	"sync/atomic"

	core_config "github.com/MIZHT/inter-exchange-arbitrage/internal/core/config"
)

type CmcIndexResponse struct {
	Values [][]interface{} `json:"values"`
}

type SymbolMap map[int]string

type MarketPairsResponse struct {
	Data struct {
		MarketPairs []MarketPair `json:"marketPairs"`
	} `json:"data"`
}

type MarketPair struct {
	ExchangeID int    `json:"exchangeId"`
	Category   string `json:"category"`
	BaseSymbol string `json:"baseSymbol"`
}

type Snapshot map[string]TickerOutput

type TickerOutput struct {
	Symbol  string                       `json:"symbol"`
	CoinID  int                          `json:"coin_id"`
	Name    string                       `json:"name"`
	Spot    [][]core_config.ExchangeName `json:"spot"`    // Группы бирж для спота
	Futures [][]core_config.ExchangeName `json:"futures"` // Группы бирж для фьючерсов
	Errors  []string                     `json:"errors,omitempty"`
}

type Store struct {
	cur atomic.Pointer[Snapshot]
}
