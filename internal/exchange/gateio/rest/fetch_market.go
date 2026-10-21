package gateio_rest

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
	spotMarketsEndpoint    = "https://api.gateio.ws/api/v4/spot/currency_pairs"
	futuresMarketsEndpoint = "https://api.gateio.ws/api/v4/futures/usdt/contracts"
)

func (c *Client) FetchMarkets(ctx context.Context) ([]domain.Market, error) {
	var (
		wg      sync.WaitGroup
		spot    []GateIOSpotMarket
		futures []GateIOFutureMarket
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
		return nil, fmt.Errorf("fetch spot pairs: %w", spotErr)
	}
	if futErr != nil {
		return nil, fmt.Errorf("fetch futures contracts: %w", futErr)
	}

	markets, err := mapMarkets(spot, futures)
	if err != nil {
		return nil, fmt.Errorf("map markets: %w", err)
	}
	c.logger.Info("GateIO FetchMarkets", slog.Int("count", len(markets)))
	return markets, nil
}

func (c *Client) fetchSpotMarket(ctx context.Context) ([]GateIOSpotMarket, error) {
	resp, err := c.httpClient.Get(ctx, spotMarketsEndpoint)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var response []GateIOSpotMarket
	if err := decodeJSON(resp, &response); err != nil {
		return nil, err
	}
	return response, nil
}

func (c *Client) fetchFuturesMarket(ctx context.Context) ([]GateIOFutureMarket, error) {
	resp, err := c.httpClient.Get(ctx, futuresMarketsEndpoint)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var response []GateIOFutureMarket
	if err := decodeJSON(resp, &response); err != nil {
		return nil, err
	}
	return response, nil
}

func mapMarkets(spot []GateIOSpotMarket, futures []GateIOFutureMarket) ([]domain.Market, error) {
	marketsByCoin := make(map[string]*domain.Market)
	for _, symbol := range spot {
		pair := strings.ToUpper(symbol.ID)
		if symbol.TradeStatus != "tradable" || !strings.HasSuffix(pair, "_USDT") {
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
		pair := strings.ToUpper(future.Name)
		if future.Status != "trading" || !strings.HasSuffix(pair, "_USDT") {
			continue
		}
		coin := strings.TrimSuffix(pair, "_USDT")
		if coin == "" {
			continue
		}
		fundingRate, err := strconv.ParseFloat(future.FundingRate, 64)
		if err != nil {
			return nil, fmt.Errorf("parse funding rate for %s: %w", future.Name, err)
		}
		market := marketsByCoin[coin]
		if market == nil {
			market = &domain.Market{Coin: coin}
			marketsByCoin[coin] = market
		}
		nextFundingTime := future.FundingNextApply
		if nextFundingTime > 0 && nextFundingTime < 1_000_000_000_000 {
			nextFundingTime *= 1000
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
