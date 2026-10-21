package bybit_rest

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
	spotMarketsEndpoint    = "https://api.bybit.com/v5/market/instruments-info?category=spot"
	futuresMarketsEndpoint = "https://api.bybit.com/v5/market/tickers?category=linear"
)

func (c *Client) FetchMarkets(ctx context.Context) ([]domain.Market, error) {
	var (
		wg      sync.WaitGroup
		spot    BybitSpotMarketResponse
		futures BybitFutureMarketResponse
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
		return nil, fmt.Errorf("fetch spot instruments: %w", spotErr)
	}
	if futErr != nil {
		return nil, fmt.Errorf("fetch futures tickers: %w", futErr)
	}

	markets, err := mapMarkets(spot.Result.List, futures.Result.List)
	if err != nil {
		return nil, fmt.Errorf("map markets: %w", err)
	}
	c.logger.Info("Bybit FetchMarkets", slog.Int("count", len(markets)))
	return markets, nil
}

func (c *Client) fetchSpotMarket(ctx context.Context) (BybitSpotMarketResponse, error) {
	resp, err := c.httpClient.Get(ctx, spotMarketsEndpoint)
	if err != nil {
		return BybitSpotMarketResponse{}, err
	}
	defer resp.Body.Close()

	var response BybitSpotMarketResponse
	if err := decodeJSON(resp, &response); err != nil {
		return BybitSpotMarketResponse{}, err
	}
	if response.RetCode != 0 {
		return BybitSpotMarketResponse{}, fmt.Errorf("Bybit API error: retCode=%d, retMsg=%s", response.RetCode, response.RetMsg)
	}
	return response, nil
}

func (c *Client) fetchFuturesMarket(ctx context.Context) (BybitFutureMarketResponse, error) {
	resp, err := c.httpClient.Get(ctx, futuresMarketsEndpoint)
	if err != nil {
		return BybitFutureMarketResponse{}, err
	}
	defer resp.Body.Close()

	var response BybitFutureMarketResponse
	if err := decodeJSON(resp, &response); err != nil {
		return BybitFutureMarketResponse{}, err
	}
	if response.RetCode != 0 {
		return BybitFutureMarketResponse{}, fmt.Errorf("Bybit API error: retCode=%d, retMsg=%s", response.RetCode, response.RetMsg)
	}
	return response, nil
}

func mapMarkets(spot []BybitSpotMarket, futures []BybitFutureMarket) ([]domain.Market, error) {
	marketsByCoin := make(map[string]*domain.Market)
	for _, symbol := range spot {
		pair := strings.ToUpper(symbol.Symbol)
		if symbol.Status != "Trading" || !strings.HasSuffix(pair, "USDT") {
			continue
		}
		coin := strings.TrimSuffix(pair, "USDT")
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
		if future.NextFundingTime == "0" || !strings.HasSuffix(pair, "USDT") {
			continue
		}
		coin := strings.TrimSuffix(pair, "USDT")
		if coin == "" {
			continue
		}
		fundingRate, err := strconv.ParseFloat(future.FundingRate, 64)
		if err != nil {
			return nil, fmt.Errorf("parse funding rate for %s: %w", future.Symbol, err)
		}
		nextFundingTime, err := strconv.ParseInt(future.NextFundingTime, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("parse next funding time for %s: %w", future.Symbol, err)
		}
		market := marketsByCoin[coin]
		if market == nil {
			market = &domain.Market{Coin: coin}
			marketsByCoin[coin] = market
		}
		market.Futurable = true
		market.FuturesFundingRate = fundingRate
		market.FuturesNextFundingTime = nextFundingTime
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
