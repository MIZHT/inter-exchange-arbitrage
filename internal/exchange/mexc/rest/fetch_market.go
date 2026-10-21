package mexc_rest

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"

	"github.com/MIZHT/inter-exchange-arbitrage/internal/core/domain"
)

const (
	spotMarketsEndpoint    = "https://api.mexc.com/api/v3/exchangeInfo"
	futuresMarketsEndpoint = "https://contract.mexc.com/api/v1/contract/funding_rate"
)

func (c *Client) FetchMarkets(ctx context.Context) ([]domain.Market, error) {
	var (
		wg      sync.WaitGroup
		spot    MEXCSpotMarketResponse
		futures MEXCFutureMarketResponse
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

	markets, err := mapMarkets(spot.Symbols, futures.Data)
	if err != nil {
		return nil, fmt.Errorf("map markets: %w", err)
	}
	c.logger.Info("MEXC FetchMarkets", slog.Int("count", len(markets)))
	return markets, nil
}

func (c *Client) fetchSpotMarket(ctx context.Context) (MEXCSpotMarketResponse, error) {
	resp, err := c.httpClient.Get(ctx, spotMarketsEndpoint)
	if err != nil {
		return MEXCSpotMarketResponse{}, err
	}
	defer resp.Body.Close()

	var response MEXCSpotMarketResponse
	if err := decodeJSON(resp, &response); err != nil {
		return MEXCSpotMarketResponse{}, err
	}
	return response, nil
}

func (c *Client) fetchFuturesMarket(ctx context.Context) (MEXCFutureMarketResponse, error) {
	resp, err := c.httpClient.Get(ctx, futuresMarketsEndpoint)
	if err != nil {
		return MEXCFutureMarketResponse{}, err
	}
	defer resp.Body.Close()

	var response MEXCFutureMarketResponse
	if err := decodeJSON(resp, &response); err != nil {
		return MEXCFutureMarketResponse{}, err
	}
	if !response.Success {
		return MEXCFutureMarketResponse{}, fmt.Errorf("MEXC API error: code=%d", response.Code)
	}
	return response, nil
}

func mapMarkets(spot []MEXCSpotMarket, futures []MEXCFutureMarket) ([]domain.Market, error) {
	marketsByCoin := make(map[string]*domain.Market)
	for _, symbol := range spot {
		if symbol.Status.String() != "1" || !symbol.IsSpotTradingAllowed || strings.ToUpper(symbol.QuoteAsset) != "USDT" {
			continue
		}
		coin := strings.ToUpper(symbol.BaseAsset)
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
		if !strings.HasSuffix(pair, "_USDT") {
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
		market.FuturesNextFundingTime = future.NextSettleTime
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
