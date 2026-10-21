package bitmart_rest

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
	spotMarketsEndpoint    = "https://api-cloud.bitmart.com/spot/v1/symbols/details"
	futuresMarketsEndpoint = "https://api-cloud-v2.bitmart.com/contract/public/details"
)

func (c *Client) FetchMarkets(ctx context.Context) ([]domain.Market, error) {
	var (
		wg      sync.WaitGroup
		spot    BitmartSpotMarketResponse
		futures BitmartFutureMarketResponse
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
		return nil, fmt.Errorf("fetch futures details: %w", futErr)
	}

	markets, err := mapMarkets(spot.Data.Symbols, futures.Data.Symbols)
	if err != nil {
		return nil, fmt.Errorf("map markets: %w", err)
	}
	c.logger.Info("BitMart FetchMarkets", slog.Int("count", len(markets)))
	return markets, nil
}

func (c *Client) fetchSpotMarket(ctx context.Context) (BitmartSpotMarketResponse, error) {
	resp, err := c.httpClient.Get(ctx, spotMarketsEndpoint)
	if err != nil {
		return BitmartSpotMarketResponse{}, err
	}
	defer resp.Body.Close()

	var response BitmartSpotMarketResponse
	if err := decodeJSON(resp, &response); err != nil {
		return BitmartSpotMarketResponse{}, err
	}
	code := fmt.Sprint(response.Code)
	if code != "1000" && code != "0" {
		return BitmartSpotMarketResponse{}, fmt.Errorf("BitMart API error: code=%s, message=%s", code, response.Message)
	}
	return response, nil
}

func (c *Client) fetchFuturesMarket(ctx context.Context) (BitmartFutureMarketResponse, error) {
	resp, err := c.httpClient.Get(ctx, futuresMarketsEndpoint)
	if err != nil {
		return BitmartFutureMarketResponse{}, err
	}
	defer resp.Body.Close()

	var response BitmartFutureMarketResponse
	if err := decodeJSON(resp, &response); err != nil {
		return BitmartFutureMarketResponse{}, err
	}
	code := fmt.Sprint(response.Code)
	if code != "1000" && code != "0" {
		return BitmartFutureMarketResponse{}, fmt.Errorf("BitMart API error: code=%s, message=%s", code, response.Message)
	}
	return response, nil
}

func mapMarkets(spot []BitmartSpotMarket, futures []BitmartFutureMarket) ([]domain.Market, error) {
	marketsByCoin := make(map[string]*domain.Market)

	for _, symbol := range spot {
		pair := strings.ToUpper(symbol.Symbol)
		if symbol.TradeStatus != "trading" || !strings.HasSuffix(pair, "_USDT") {
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
		market.Spotable = true
	}

	for _, future := range futures {
		if future.Status != "Trading" {
			continue
		}
		pair := strings.ToUpper(future.Symbol)
		coin := ""
		switch {
		case strings.HasSuffix(pair, "_USDT"):
			coin = strings.TrimSuffix(pair, "_USDT")
		case strings.HasSuffix(pair, "-USDT"):
			coin = strings.TrimSuffix(pair, "-USDT")
		case strings.HasSuffix(pair, "USDT"):
			coin = strings.TrimSuffix(pair, "USDT")
		}
		if coin == "" {
			continue
		}
		rate, err := strconv.ParseFloat(future.FundingRate, 64)
		if err != nil {
			return nil, fmt.Errorf("parse funding rate for %s: %w", future.Symbol, err)
		}
		market := marketsByCoin[coin]
		if market == nil {
			market = &domain.Market{Coin: coin}
			marketsByCoin[coin] = market
		}
		market.Futurable = true
		market.FuturesFundingRate = rate
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
