package domain

import (
	"sync/atomic"

	core_config "github.com/MIZHT/inter-exchange-arbitrage/internal/core/config"
)

type MatchStore struct {
	Cur atomic.Pointer[MatchResult]
}

type MatchResult struct {
	Spot    map[core_config.ExchangeName]map[string]struct{}
	Futures map[core_config.ExchangeName]map[string]struct{}
	Matches map[string]*ContractMatch
}

type ContractMatch struct {
	Coin      string            `json:"coin"`
	Exchanges []MatchedExchange `json:"exchanges"`
	Pairs     []ExchangePair    `json:"pairs"`
}

type MatchedExchange struct {
	Name                   string  `json:"name"`
	FuturesFundingRate     float64 `json:"futures_funding_rate,omitempty"`
	FuturesNextFundingTime int64   `json:"futures_next_funding_time,omitempty"`
}

type ExchangePair struct {
	WithdrawExchange   string  `json:"withdraw_exchange"`
	DepositExchange    string  `json:"deposit_exchange"`
	NetworkName        string  `json:"network_name,omitempty"`
	WithdrawFee        float64 `json:"withdraw_fee,omitempty"`
	AvailableDirection string  `json:"available_direction"`
}

func NewMatches() *MatchStore {
	s := &MatchStore{}
	empty := MatchResult{}
	s.Cur.Store(&empty)
	return s
}

func (s *MatchStore) Get() *MatchResult { return s.Cur.Load() }

func (s *MatchStore) Publish(snap *MatchResult) { s.Cur.Store(snap) }
