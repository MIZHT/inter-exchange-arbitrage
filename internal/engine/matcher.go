package engine

import (
	"log/slog"
	"time"

	core_config "github.com/MIZHT/inter-exchange-arbitrage/internal/core/config"
	"github.com/MIZHT/inter-exchange-arbitrage/internal/core/domain"
)

func (c *Coordinator) RunMatchFind(
	input MatchInput,
) {
	startedAt := time.Now()

	result := domain.MatchResult{
		Spot:    make(map[core_config.ExchangeName]map[string]struct{}),
		Futures: make(map[core_config.ExchangeName]map[string]struct{}),
		Matches: make(map[string]*domain.ContractMatch),
	}

	spotSets := make(map[core_config.ExchangeName]map[string]struct{})
	futuresSets := make(map[core_config.ExchangeName]map[string]struct{})

	for coin, cmcInfo := range *input.CMC {
		match := &domain.ContractMatch{Coin: coin}

		exchangeIndexes := make(map[core_config.ExchangeName]int)
		matched := false

		// spot - spot
		for _, spotGroup := range cmcInfo.Spot {
			for i := 0; i < len(spotGroup); i++ {
				ex1 := spotGroup[i]
				exchange1 := input.Exchanges[ex1]
				if exchange1 == nil {
					continue
				}

				exInfo1, ok := exchange1.FilteredPairs[coin]
				if !ok || !exInfo1.SpotStatus.Available {
					continue
				}

				for j := i + 1; j < len(spotGroup); j++ {
					ex2 := spotGroup[j]
					if ex1 == ex2 {
						continue
					}

					exchange2 := input.Exchanges[ex2]
					if exchange2 == nil {
						continue
					}

					exInfo2, ok := exchange2.FilteredPairs[coin]
					if !ok || !exInfo2.SpotStatus.Available {
						continue
					}

					// Проверяем оба направления перевода.
					routes := findSpotRoutes(ex1, ex2, exInfo1, exInfo2)
					routes = append(
						routes,
						findSpotRoutes(ex2, ex1, exInfo2, exInfo1)...,
					)
					if len(routes) == 0 {
						continue
					}

					matched = true
					addMatchedExchange(match, exchangeIndexes, ex1, exInfo1)
					addMatchedExchange(match, exchangeIndexes, ex2, exInfo2)

					match.Pairs = append(match.Pairs, routes...)

					addCoin(spotSets, ex1, coin)
					addCoin(spotSets, ex2, coin)
				}
			}
		}

		// spot - futures
		groupCount := len(cmcInfo.Spot)
		if len(cmcInfo.Futures) < groupCount {
			groupCount = len(cmcInfo.Futures)
		}

		for groupIndex := 0; groupIndex < groupCount; groupIndex++ {
			spotGroup := cmcInfo.Spot[groupIndex]
			futuresGroup := cmcInfo.Futures[groupIndex]

			for _, spotEx := range spotGroup {
				spotExchange := input.Exchanges[spotEx]
				if spotExchange == nil {
					continue
				}

				spotInfo, ok := spotExchange.FilteredPairs[coin]
				if !ok || !spotInfo.SpotStatus.Available {
					continue
				}

				for _, futuresEx := range futuresGroup {
					if spotEx == futuresEx {
						continue
					}

					futuresExchange := input.Exchanges[futuresEx]
					if futuresExchange == nil {
						continue
					}

					futuresInfo, ok := futuresExchange.FilteredPairs[coin]
					if !ok || !futuresInfo.FutureStatus.Available {
						continue
					}

					matched = true
					addMatchedExchange(match, exchangeIndexes, spotEx, spotInfo)
					addMatchedExchange(match, exchangeIndexes, futuresEx, futuresInfo)

					match.Pairs = append(match.Pairs, domain.ExchangePair{
						WithdrawExchange:   string(spotEx),
						DepositExchange:    string(futuresEx),
						AvailableDirection: "spot_futures",
					})

					addCoin(spotSets, spotEx, coin)
					addCoin(futuresSets, futuresEx, coin)
				}
			}

		}

		// future - future
		for _, futuresGroup := range cmcInfo.Futures {
			for i := 0; i < len(futuresGroup); i++ {
				ex1 := futuresGroup[i]
				exchange1 := input.Exchanges[ex1]
				if exchange1 == nil {
					continue
				}

				exInfo1, ok := exchange1.FilteredPairs[coin]
				if !ok || !exInfo1.FutureStatus.Available {
					continue
				}

				for j := i + 1; j < len(futuresGroup); j++ {
					ex2 := futuresGroup[j]

					exchange2 := input.Exchanges[ex2]
					if exchange2 == nil {
						continue
					}

					exInfo2, ok := exchange2.FilteredPairs[coin]
					if !ok || !exInfo2.FutureStatus.Available {
						continue
					}

					matched = true
					addMatchedExchange(match, exchangeIndexes, ex1, exInfo1)
					addMatchedExchange(match, exchangeIndexes, ex2, exInfo2)

					match.Pairs = append(match.Pairs,
						domain.ExchangePair{
							WithdrawExchange:   string(ex1),
							DepositExchange:    string(ex2),
							AvailableDirection: "futures_futures",
						}, domain.ExchangePair{
							WithdrawExchange:   string(ex2),
							DepositExchange:    string(ex1),
							AvailableDirection: "futures_futures",
						})

					addCoin(futuresSets, ex1, coin)
					addCoin(futuresSets, ex2, coin)
				}
			}
		}

		if matched {
			result.Matches[coin] = match
		}
	}
	result.Spot = spotSets
	result.Futures = futuresSets
	c.matcherStore.Publish(&result)

	c.logger.Info(
		"Match completed",
		slog.Int("coins_matched", len(result.Matches)),
		slog.Int("spot_exсhanges", len(result.Spot)),
		slog.Int("futures_exchanges", len(result.Futures)),
		slog.Duration("duration", time.Since(startedAt)),
	)
}

func findSpotRoutes(
	withdrawExchange core_config.ExchangeName,
	depositExchange core_config.ExchangeName,
	withdrawPair domain.FilteredPair,
	depositPair domain.FilteredPair,
) []domain.ExchangePair {
	var routes []domain.ExchangePair

	for _, withdrawNetwork := range withdrawPair.WithdrawNetworks {
		for _, depositNetwork := range depositPair.DepositNetworks {

			if withdrawNetwork.ContractAddress != depositNetwork.ContractAddress {
				continue
			}

			routes = append(routes, domain.ExchangePair{
				WithdrawExchange:   string(withdrawExchange),
				DepositExchange:    string(depositExchange),
				NetworkName:        withdrawNetwork.Name,
				WithdrawFee:        withdrawNetwork.WithdrawFee,
				AvailableDirection: "spot_spot",
			})
		}
	}

	return routes
}

func addMatchedExchange(
	match *domain.ContractMatch,
	indexes map[core_config.ExchangeName]int,
	exchangeName core_config.ExchangeName,
	pair domain.FilteredPair,
) {
	if index, ok := indexes[exchangeName]; ok {
		updateFunding(&match.Exchanges[index], pair)
		return
	}

	exchange := domain.MatchedExchange{
		Name: string(exchangeName),
	}
	updateFunding(&exchange, pair)

	indexes[exchangeName] = len(match.Exchanges)
	match.Exchanges = append(match.Exchanges, exchange)
}

func updateFunding(
	exchange *domain.MatchedExchange,
	pair domain.FilteredPair,
) {
	if !pair.FutureStatus.Available {
		return
	}

	exchange.FuturesFundingRate = pair.FutureStatus.FundingRate
	exchange.FuturesNextFundingTime = pair.FutureStatus.NextFundingTime
}

func addCoin(
	sets map[core_config.ExchangeName]map[string]struct{},
	exchange core_config.ExchangeName,
	coin string,
) {
	if sets[exchange] == nil {
		sets[exchange] = make(map[string]struct{})
	}
	sets[exchange][coin] = struct{}{}
}
