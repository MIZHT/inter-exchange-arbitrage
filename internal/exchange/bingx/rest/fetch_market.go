package bingx_rest

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"sync"

	"github.com/MIZHT/inter-exchange-arbitrage/internal/core/domain"
)

const (
	spotMarketsEndpoint    = "https://open-api.bingx.com/openApi/spot/v1/common/symbols"
	futuresMarketsEndpoint = "https://open-api.bingx.com/openApi/swap/v2/quote/premiumIndex"
)

func (c *Client) FetchMarkets(ctx context.Context) ([]domain.Market, error) {
	var (
		wg      sync.WaitGroup
		spot    BingXSpotMarketResponse
		futures BingXFutureMarketResponse
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
		return nil, fmt.Errorf("fetch spot markets: %w", spotErr)
	}
	if futErr != nil {
		return nil, fmt.Errorf("fetch futures markets: %w", futErr)
	}

	markets, err := mapMarkets(spot, futures)
	if err != nil {
		return nil, fmt.Errorf("map markets: %w", err)
	}
	c.logger.Info("BingX FetchMarkets", slog.Int("count", len(markets)))
	return markets, nil
}

func (c *Client) fetchSpotMarket(ctx context.Context) (BingXSpotMarketResponse, error) {
	resp, err := c.httpClient.Get(ctx, spotMarketsEndpoint)
	if err != nil {
		return BingXSpotMarketResponse{}, err
	}
	defer resp.Body.Close()

	var response BingXSpotMarketResponse
	if err := decodeJSON(resp, &response); err != nil {
		return BingXSpotMarketResponse{}, err
	}
	if response.Code != 0 {
		return BingXSpotMarketResponse{}, fmt.Errorf("BingX API error: code=%d, message=%s", response.Code, response.Msg)
	}
	return response, nil
}

func (c *Client) fetchFuturesMarket(ctx context.Context) (BingXFutureMarketResponse, error) {
	resp, err := c.httpClient.Get(ctx, futuresMarketsEndpoint)
	if err != nil {
		return BingXFutureMarketResponse{}, err
	}
	defer resp.Body.Close()

	var response BingXFutureMarketResponse
	if err := decodeJSON(resp, &response); err != nil {
		return BingXFutureMarketResponse{}, err
	}
	if response.Code != 0 {
		return BingXFutureMarketResponse{}, fmt.Errorf("BingX API error: code=%d, message=%s", response.Code, response.Msg)
	}
	return response, nil
}

func mapMarkets(spot BingXSpotMarketResponse, futures BingXFutureMarketResponse) ([]domain.Market, error) {
	marketsByCoin := make(map[string]*domain.Market)
	for _, symbol := range spot.Data.Symbols {
		if symbol.Status.String() != "1" || !strings.HasSuffix(strings.ToUpper(symbol.Symbol), "-USDT") {
			continue
		}
		coin := strings.TrimSuffix(strings.ToUpper(symbol.Symbol), "-USDT")
		market := marketsByCoin[coin]
		if market == nil {
			market = &domain.Market{Coin: coin}
			marketsByCoin[coin] = market
		}
		market.Spotable = true
	}

	for _, future := range futures.Data {
		symbol := strings.ToUpper(future.Symbol)
		if !strings.HasSuffix(symbol, "-USDT") {
			continue
		}
		coin := strings.TrimSuffix(symbol, "-USDT")
		market := marketsByCoin[coin]
		if market == nil {
			market = &domain.Market{Coin: coin}
			marketsByCoin[coin] = market
		}
		rate, err := strconv.ParseFloat(future.LastFundingRate, 64)
		if err != nil {
			return nil, fmt.Errorf("parse funding rate for %s: %w", future.Symbol, err)
		}
		market.Futurable = true
		market.FuturesFundingRate = rate
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
