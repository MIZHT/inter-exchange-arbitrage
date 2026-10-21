package xt_rest

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"strings"
	"sync"

	"github.com/MIZHT/inter-exchange-arbitrage/internal/core/domain"
)

const (
	spotMarketsEndpoint    = baseURL + "/v4/public/symbol"
	futuresFundingEndpoint = "https://fapi.xt.com/future/market/v1/public/q/funding-rate?symbol=%s"
)

func (c *Client) FetchMarkets(ctx context.Context) ([]domain.Market, error) {
	spot, err := c.fetchSpotMarket(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetch spot symbols: %w", err)
	}
	futures, err := c.fetchFuturesMarket(ctx, spot.Result.Symbols)
	if err != nil {
		return nil, fmt.Errorf("fetch futures funding: %w", err)
	}
	markets, err := mapMarkets(spot.Result.Symbols, futures)
	if err != nil {
		return nil, fmt.Errorf("map markets: %w", err)
	}
	c.logger.Info("XT FetchMarkets", slog.Int("count", len(markets)))
	return markets, nil
}

func (c *Client) fetchSpotMarket(ctx context.Context) (XTSpotMarketResponse, error) {
	resp, err := c.httpClient.Get(ctx, spotMarketsEndpoint)
	if err != nil {
		return XTSpotMarketResponse{}, err
	}
	defer resp.Body.Close()

	var response XTSpotMarketResponse
	if err := decodeJSON(resp, &response); err != nil {
		return XTSpotMarketResponse{}, err
	}
	if response.ReturnCode != 0 {
		return XTSpotMarketResponse{}, fmt.Errorf("XT API error: %s (code %d)", response.Message, response.ReturnCode)
	}
	return response, nil
}

func (c *Client) fetchFuturesMarket(ctx context.Context, spot []XTSpotMarket) ([]XTFutureMarket, error) {
	futures := make([]XTFutureMarket, len(spot))
	sem := make(chan struct{}, 4)
	var wg sync.WaitGroup
	for i, symbol := range spot {
		if !symbol.TradingEnabled || !symbol.OpenAPIEnabled || !strings.EqualFold(symbol.State, "online") || strings.ToUpper(symbol.QuoteCurrency) != "USDT" || symbol.BaseCurrency == "" {
			continue
		}
		wg.Add(1)
		go func(i int, symbol XTSpotMarket) {
			defer wg.Done()
			select {
			case sem <- struct{}{}:
			case <-ctx.Done():
				return
			}
			defer func() { <-sem }()

			baseCoin := strings.ToUpper(symbol.BaseCurrency)
			requestSymbol := strings.ToLower(baseCoin + "_USDT")
			endpoint := fmt.Sprintf(futuresFundingEndpoint, url.QueryEscape(requestSymbol))
			resp, err := c.httpClient.Get(ctx, endpoint)
			if err != nil {
				return
			}
			defer resp.Body.Close()

			var response XTFutureMarketResponse
			if decodeJSON(resp, &response) != nil || response.ReturnCode != 0 || response.Result.NextFundingTime <= 0 {
				return
			}
			futures[i] = response.Result
		}(i, symbol)
	}
	wg.Wait()
	result := make([]XTFutureMarket, 0, len(futures))
	for _, future := range futures {
		if future.Symbol != "" {
			result = append(result, future)
		}
	}
	return result, nil
}

func mapMarkets(spot []XTSpotMarket, futures []XTFutureMarket) ([]domain.Market, error) {
	marketsByCoin := make(map[string]*domain.Market)
	for _, symbol := range spot {
		if !symbol.TradingEnabled || !symbol.OpenAPIEnabled || !strings.EqualFold(symbol.State, "online") || strings.ToUpper(symbol.QuoteCurrency) != "USDT" {
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
		pair := strings.ToUpper(future.Symbol)
		if !strings.HasSuffix(pair, "_USDT") || future.NextFundingTime <= 0 {
			continue
		}
		coin := strings.TrimSuffix(pair, "_USDT")
		if coin == "" {
			continue
		}
		market := marketsByCoin[coin]
		if market == nil {
			market = &domain.Market{Coin: coin}
			marketsByCoin[coin] = market
		}
		market.Futurable = true
		market.FuturesFundingRate = future.FundingRate
		market.FuturesNextFundingTime = future.NextFundingTime
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
