package bitget_rest

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
	spotMarketsEndpoint    = "https://api.bitget.com/api/v2/spot/public/symbols"
	futuresMarketsEndpoint = "https://api.bitget.com/api/v2/mix/market/current-fund-rate?productType=USDT-FUTURES"
)

func (c *Client) FetchMarkets(ctx context.Context) ([]domain.Market, error) {
	var (
		wg      sync.WaitGroup
		spot    BitgetSpotMarketResponse
		futures BitgetFutureMarketResponse
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
		return nil, fmt.Errorf("fetch futures funding: %w", futErr)
	}

	markets, err := mapMarkets(spot.Data, futures.Data)
	if err != nil {
		return nil, fmt.Errorf("map markets: %w", err)
	}
	c.logger.Info("Bitget FetchMarkets", slog.Int("count", len(markets)))
	return markets, nil
}

func (c *Client) fetchSpotMarket(ctx context.Context) (BitgetSpotMarketResponse, error) {
	resp, err := c.httpClient.Get(ctx, spotMarketsEndpoint)
	if err != nil {
		return BitgetSpotMarketResponse{}, err
	}
	defer resp.Body.Close()

	var response BitgetSpotMarketResponse
	if err := decodeJSON(resp, &response); err != nil {
		return BitgetSpotMarketResponse{}, err
	}
	if response.Code != "00000" {
		return BitgetSpotMarketResponse{}, fmt.Errorf("Bitget API error: code=%s, message=%s", response.Code, response.Msg)
	}
	return response, nil
}

func (c *Client) fetchFuturesMarket(ctx context.Context) (BitgetFutureMarketResponse, error) {
	resp, err := c.httpClient.Get(ctx, futuresMarketsEndpoint)
	if err != nil {
		return BitgetFutureMarketResponse{}, err
	}
	defer resp.Body.Close()

	var response BitgetFutureMarketResponse
	if err := decodeJSON(resp, &response); err != nil {
		return BitgetFutureMarketResponse{}, err
	}
	if response.Code != "00000" {
		return BitgetFutureMarketResponse{}, fmt.Errorf("Bitget API error: code=%s, message=%s", response.Code, response.Msg)
	}
	return response, nil
}

func mapMarkets(spot []BitgetSpotMarket, futures []BitgetFutureMarket) ([]domain.Market, error) {
	marketsByCoin := make(map[string]*domain.Market)

	for _, symbol := range spot {
		pair := strings.ToUpper(symbol.Symbol)
		if symbol.Status != "online" || !strings.HasSuffix(pair, "USDT") {
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
		if !strings.HasSuffix(pair, "USDT") {
			continue
		}
		coin := strings.TrimSuffix(pair, "USDT")
		if coin == "" {
			continue
		}
		rate, err := strconv.ParseFloat(future.FundingRate, 64)
		if err != nil {
			return nil, fmt.Errorf("parse funding rate for %s: %w", future.Symbol, err)
		}
		nextFundingTime, err := strconv.ParseInt(future.NextUpdate, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("parse next funding time for %s: %w", future.Symbol, err)
		}
		market := marketsByCoin[coin]
		if market == nil {
			market = &domain.Market{Coin: coin}
			marketsByCoin[coin] = market
		}
		market.Futurable = true
		market.FuturesFundingRate = rate
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
