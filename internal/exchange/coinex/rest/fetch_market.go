package coinex_rest

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
	spotMarketsEndpoint   = "https://api.coinex.com/v2/spot/market"
	futuresTickerEndpoint = "https://api.coinex.com/perpetual/v1/market/ticker/all"
)

func (c *Client) FetchMarkets(ctx context.Context) ([]domain.Market, error) {
	var (
		wg      sync.WaitGroup
		spot    CoinExSpotMarketResponse
		futures CoinExFutureMarketResponse
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
		return nil, fmt.Errorf("fetch futures tickers: %w", futErr)
	}

	markets, err := mapMarkets(spot.Data, futures.Data.Ticker)
	if err != nil {
		return nil, fmt.Errorf("map markets: %w", err)
	}
	c.logger.Info("CoinEx FetchMarkets", slog.Int("count", len(markets)))
	return markets, nil
}

func (c *Client) fetchSpotMarket(ctx context.Context) (CoinExSpotMarketResponse, error) {
	resp, err := c.httpClient.Get(ctx, spotMarketsEndpoint)
	if err != nil {
		return CoinExSpotMarketResponse{}, err
	}
	defer resp.Body.Close()

	var response CoinExSpotMarketResponse
	if err := decodeJSON(resp, &response); err != nil {
		return CoinExSpotMarketResponse{}, err
	}
	if response.Code != 0 {
		return CoinExSpotMarketResponse{}, fmt.Errorf("CoinEx API error: code=%d, message=%s", response.Code, response.Message)
	}
	return response, nil
}

func (c *Client) fetchFuturesMarket(ctx context.Context) (CoinExFutureMarketResponse, error) {
	resp, err := c.httpClient.Get(ctx, futuresTickerEndpoint)
	if err != nil {
		return CoinExFutureMarketResponse{}, err
	}
	defer resp.Body.Close()

	var response CoinExFutureMarketResponse
	if err := decodeJSON(resp, &response); err != nil {
		return CoinExFutureMarketResponse{}, err
	}
	if response.Code != 0 {
		return CoinExFutureMarketResponse{}, fmt.Errorf("CoinEx API error: code=%d, message=%s", response.Code, response.Message)
	}
	return response, nil
}

func mapMarkets(spot []CoinExSpotMarket, futures map[string]CoinExFutureMarket) ([]domain.Market, error) {
	marketsByCoin := make(map[string]*domain.Market)
	for _, symbol := range spot {
		if symbol.Status != "online" || !symbol.IsAPITradingAvailable || strings.ToUpper(symbol.QuoteCcy) != "USDT" {
			continue
		}
		coin := strings.ToUpper(symbol.BaseCcy)
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

	for symbol, future := range futures {
		pair := strings.ToUpper(symbol)
		if future.FundingTime == 0 || !strings.HasSuffix(pair, "USDT") {
			continue
		}
		coin := strings.TrimSuffix(pair, "USDT")
		if coin == "" {
			continue
		}
		fundingRate, err := strconv.ParseFloat(future.FundingRateNext, 64)
		if err != nil {
			return nil, fmt.Errorf("parse funding rate for %s: %w", symbol, err)
		}
		market := marketsByCoin[coin]
		if market == nil {
			market = &domain.Market{Coin: coin}
			marketsByCoin[coin] = market
		}
		market.Futurable = true
		market.FuturesFundingRate = fundingRate
		market.FuturesNextFundingTime = future.FundingTime
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
