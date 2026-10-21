package kucoin_rest

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"

	"github.com/MIZHT/inter-exchange-arbitrage/internal/core/domain"
)

const (
	spotMarketsEndpoint    = "https://api.kucoin.com/api/v1/symbols"
	futuresMarketsEndpoint = "https://api-futures.kucoin.com/api/v1/contracts/active"
)

func (c *Client) FetchMarkets(ctx context.Context) ([]domain.Market, error) {
	var (
		wg      sync.WaitGroup
		spot    KuCoinSpotMarketResponse
		futures KuCoinFutureMarketResponse
		spotErr error
		futErr  error
	)
	wg.Add(2)
	go func() {
		defer wg.Done()
		spot, spotErr = c.fetchSpotMarket(ctx)
	}()
	go func() {
		defer wg.Done()
		futures, futErr = c.fetchFuturesMarket(ctx)
	}()
	wg.Wait()
	if spotErr != nil {
		return nil, fmt.Errorf("fetch spot symbols: %w", spotErr)
	}
	if futErr != nil {
		return nil, fmt.Errorf("fetch futures contracts: %w", futErr)
	}

	markets, err := mapMarkets(spot.Data, futures.Data)
	if err != nil {
		return nil, fmt.Errorf("map markets: %w", err)
	}
	c.logger.Info("KuCoin FetchMarkets", slog.Int("count", len(markets)))
	return markets, nil
}

func (c *Client) fetchSpotMarket(ctx context.Context) (KuCoinSpotMarketResponse, error) {
	resp, err := c.httpClient.Get(ctx, spotMarketsEndpoint)
	if err != nil {
		return KuCoinSpotMarketResponse{}, err
	}
	defer resp.Body.Close()

	var response KuCoinSpotMarketResponse
	if err := decodeJSON(resp, &response); err != nil {
		return KuCoinSpotMarketResponse{}, err
	}
	if response.Code != "200000" {
		return KuCoinSpotMarketResponse{}, fmt.Errorf("KuCoin API error: code=%s", response.Code)
	}
	return response, nil
}

func (c *Client) fetchFuturesMarket(ctx context.Context) (KuCoinFutureMarketResponse, error) {
	resp, err := c.httpClient.Get(ctx, futuresMarketsEndpoint)
	if err != nil {
		return KuCoinFutureMarketResponse{}, err
	}
	defer resp.Body.Close()

	var response KuCoinFutureMarketResponse
	if err := decodeJSON(resp, &response); err != nil {
		return KuCoinFutureMarketResponse{}, err
	}
	if response.Code != "200000" {
		return KuCoinFutureMarketResponse{}, fmt.Errorf("KuCoin API error: code=%s", response.Code)
	}
	return response, nil
}

func mapMarkets(spot []KuCoinSpotMarket, futures []KuCoinFutureMarket) ([]domain.Market, error) {
	marketsByCoin := make(map[string]*domain.Market)
	for _, symbol := range spot {
		pair := strings.ToUpper(symbol.Symbol)
		if !symbol.EnableTrading || strings.ToUpper(symbol.QuoteCurrency) != "USDT" || !strings.HasSuffix(pair, "-USDT") {
			continue
		}
		coin := strings.ToUpper(symbol.BaseCurrency)
		if coin == "" {
			continue
		}
		market := marketsByCoin[coin]
		if market == nil {
			market = &domain.Market{Coin: coin}
			marketsByCoin[coin] = market
		}
		market.Spotable = true
	}

	for _, future := range futures {
		if future.Status != "Open" || strings.ToUpper(future.QuoteCurrency) != "USDT" || strings.ToUpper(future.SettleCurrency) != "USDT" {
			continue
		}
		coin := strings.ToUpper(future.BaseCurrency)
		if coin == "XBT" {
			coin = "BTC"
		}
		if coin == "" {
			continue
		}
		market := marketsByCoin[coin]
		if market == nil {
			market = &domain.Market{Coin: coin}
			marketsByCoin[coin] = market
		}
		market.Futurable = true
		market.FuturesFundingRate = future.FundingFeeRate
		market.FuturesNextFundingTime = future.NextFundingRateDateTime
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
