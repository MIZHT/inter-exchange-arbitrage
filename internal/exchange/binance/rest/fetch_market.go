package binance_rest

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/MIZHT/inter-exchange-arbitrage/internal/core/domain"
	"go.uber.org/zap"
)

func (c *Client) FetchMarkets(ctx context.Context) ([]domain.Market, error) {
	var (
		wg      sync.WaitGroup
		spots   BinanceExchangeInfo
		futures []BinanceFutureMarket
		spotErr error
		futErr  error
	)

	wg.Add(2)
	go func() {
		defer wg.Done()
		spots, spotErr = c.fetchSpotMarket(ctx)
	}()
	go func() {
		defer wg.Done()
		futures, futErr = c.fetchFuturesMarket(ctx)
	}()
	wg.Wait()

	if spotErr != nil {
		return nil, fmt.Errorf("fetch spot exchange info: %w", spotErr)
	}
	if futErr != nil {
		return nil, fmt.Errorf("fetch futures premium index: %w", futErr)
	}

	markets, err := mapMarkets(spots, futures)
	if err != nil {
		return nil, fmt.Errorf("map markets: %w", err)
	}

	c.logger.Info("Binance FetchMarkets", zap.Int("count", len(markets)))
	return markets, nil
}

func (c *Client) fetchSpotMarket(ctx context.Context) (BinanceExchangeInfo, error) {
	resp, err := c.httpClient.Get(ctx, spotMarketsEndpoint)
	if err != nil {
		return BinanceExchangeInfo{}, err
	}
	defer resp.Body.Close()

	var info BinanceExchangeInfo
	if err := decodeJSON(resp, &info); err != nil {
		return BinanceExchangeInfo{}, fmt.Errorf("decode spot exchange info: %w", err)
	}
	return info, nil
}

func (c *Client) fetchFuturesMarket(ctx context.Context) ([]BinanceFutureMarket, error) {
	resp, err := c.httpClient.Get(ctx, futuresMarketsEndpoint)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var futures []BinanceFutureMarket
	if err := decodeJSON(resp, &futures); err != nil {
		return nil, fmt.Errorf("decode futures premium index: %w", err)
	}
	return futures, nil
}

func mapMarkets(
	spot BinanceExchangeInfo,
	future []BinanceFutureMarket,
) ([]domain.Market, error) {
	marketsByCoin := make(map[string]*domain.Market)

	for _, s := range spot.Symbols {
		if s.QuoteAsset != "USDT" {
			continue
		}
		if s.Status != "TRADING" || !s.IsSpotTradingAllowed {
			continue
		}

		coin := strings.ToUpper(s.BaseAsset)
		getOrCreateMarket(marketsByCoin, coin).Spotable = true
	}

	for _, p := range future {
		if p.NextFundingTime <= 0 {
			continue
		}
		if !strings.HasSuffix(p.Symbol, "USDT") {
			continue
		}

		rate, err := strconv.ParseFloat(p.LastFundingRate, 64)
		if err != nil {
			return nil, fmt.Errorf("parse funding rate for %s: %w", p.Symbol, err)
		}

		coin := strings.TrimSuffix(p.Symbol, "USDT")
		market := getOrCreateMarket(marketsByCoin, coin)
		market.Futurable = true
		market.FuturesFundingRate = rate
		market.FuturesNextFundingTime = p.NextFundingTime
	}

	markets := make([]domain.Market, 0, len(marketsByCoin))
	for _, market := range marketsByCoin {
		if !market.Spotable && !market.Futurable {
			continue
		}
		markets = append(markets, *market)
	}
	return markets, nil
}

func getOrCreateMarket(markets map[string]*domain.Market, coin string) *domain.Market {
	market := markets[coin]
	if market == nil {
		market = &domain.Market{Coin: coin}
		markets[coin] = market
	}
	return market
}
