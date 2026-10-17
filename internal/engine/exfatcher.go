package engine

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"

	core_config "github.com/MIZHT/inter-exchange-arbitrage/internal/core/config"
	"github.com/MIZHT/inter-exchange-arbitrage/internal/core/domain"
	core_logger "github.com/MIZHT/inter-exchange-arbitrage/internal/core/logger"
	"github.com/MIZHT/inter-exchange-arbitrage/internal/exchange"
)

type ExchangeRunner struct {
	Exchange core_config.ExchangeName

	Adapter exchange.ExchangeAdapter
	logger  *core_logger.Logger

	snapshot atomic.Pointer[ExchangeSnapshot]

	mu sync.Mutex
}

type ExchangeSnapshot struct {
	FilteredPairs domain.FilteredPairs
	SpotCoins     map[string]struct{}
	FutureCoins   map[string]struct{}
}

func NewExchangeRunner(
	adapter exchange.ExchangeAdapter,
	logger *core_logger.Logger,
) *ExchangeRunner {
	return &ExchangeRunner{
		Adapter: adapter,
		logger:  logger,
	}
}

func (r *ExchangeRunner) RunExFetch(ctx context.Context) error {
	var (
		wg         sync.WaitGroup
		currencies []domain.Currency
		markets    []domain.Market
		errs       = make(chan error, 2)
	)
	exchangeName := r.Adapter.Name()

	wg.Add(2)
	go func() {
		defer wg.Done()
		c, err := r.Adapter.FetchCurrencies(ctx)
		if err != nil {
			errs <- fmt.Errorf("ex: %s: fetch currencies: %w", exchangeName, err)
			return
		}
		currencies = c
	}()
	go func() {
		defer wg.Done()
		m, err := r.Adapter.FetchMarkets(ctx)
		if err != nil {
			errs <- fmt.Errorf("ex: %s: fetch markets: %w", exchangeName, err)
			return
		}
		markets = m
	}()

	wg.Wait()
	close(errs)

	for err := range errs {
		if err != nil {
			return err
		}
	}
	r.filterPairs(currencies, markets)

	return nil
}

func (r *ExchangeRunner) filterPairs(
	currenciesSlice []domain.Currency,
	marketsSlice []domain.Market,
) {

	fp := make(domain.FilteredPairs)
	spotCoins := make(map[string]struct{})
	futureCoins := make(map[string]struct{})

	marketsMap := collapseMarkets(marketsSlice)

	for _, currency := range currenciesSlice {
		var filteredPair domain.FilteredPair

		market, ok := marketsMap[currency.Name]
		if !ok {
			continue
		}

		if market.Futurable {
			futureCoins[market.Coin] = struct{}{}
			filteredPair.FutureStatus = domain.MarketStatus{
				Available:       market.Futurable,
				FundingRate:     market.FuturesFundingRate,
				NextFundingTime: market.FuturesNextFundingTime,
			}
		}
		if market.Spotable {
			spotCoins[market.Coin] = struct{}{}
			filteredPair.SpotStatus = domain.MarketStatus{
				Available: market.Spotable,
			}
		}

		for _, network := range currency.Networks {
			if network.WithdrawEnabled {
				filteredPair.WithdrawNetworks = append(filteredPair.WithdrawNetworks,
					domain.NetworkInfo{
						Name:            network.Name,
						ContractAddress: network.ContractAddress,
						WithdrawFee:     network.WithdrawFee,
					})
			}
			if network.DepositEnabled {
				filteredPair.DepositNetworks = append(filteredPair.DepositNetworks,
					domain.NetworkInfo{
						Name:            network.Name,
						ContractAddress: network.ContractAddress,
					})
			}
		}
		fp[currency.Name] = filteredPair
	}

	r.snapshot.Store(&ExchangeSnapshot{
		FilteredPairs: fp,
		SpotCoins:     spotCoins,
		FutureCoins:   futureCoins,
	})

}

func collapseMarkets(markets []domain.Market) map[string]domain.Market {
	result := make(map[string]domain.Market, len(markets))
	for _, market := range markets {
		if market.Coin == "" {
			continue
		}
		current := result[market.Coin]
		current.Coin = market.Coin
		current.Spotable = market.Spotable
		current.Futurable = market.Futurable
		current.FuturesFundingRate = market.FuturesFundingRate
		current.FuturesNextFundingTime = market.FuturesNextFundingTime
		result[market.Coin] = current
	}
	return result
}
