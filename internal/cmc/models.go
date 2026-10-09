package cmc

import (
	"sync/atomic"
)

type CmcIndexResponse struct {
	Values [][]interface{} `json:"values"`
}

type SymbolMap map[int]string

// Ответ API
type MarketPairsResponse struct {
	Data struct {
		MarketPairs []MarketPair `json:"marketPairs"`
	} `json:"data"`
}

// Промежуточная структура для marketPair
type MarketPair struct {
	ExchangeID int    `json:"exchangeId"`
	Category   string `json:"category"`
	BaseSymbol string `json:"baseSymbol"`
}

// Карта: symbol -> TickerOutput
type Snapshot map[string]TickerOutput

// Вывод для одного тикера
type TickerOutput struct {
	Symbol  string     `json:"symbol"`
	CoinID  int        `json:"coin_id"`
	Name    string     `json:"name"`
	Spot    [][]string `json:"spot"`    // Группы бирж для спота
	Futures [][]string `json:"futures"` // Группы бирж для фьючерсов
	Errors  []string   `json:"errors,omitempty"`
}

type Store struct {
	cur atomic.Pointer[Snapshot]
}
