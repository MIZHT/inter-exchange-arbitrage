package okx_rest

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"strconv"
	"strings"
	"sync"

	"github.com/MIZHT/inter-exchange-arbitrage/internal/core/domain"
)

const (
	spotMarketsEndpoint    = "https://www.okx.com/api/v5/public/instruments?instType=SPOT"
	futuresMarketsEndpoint = "https://www.okx.com/api/v5/public/instruments?instType=SWAP"
	fundingEndpointFormat  = "https://www.okx.com/api/v5/public/funding-rate?instId=%s"
)

func (c *Client) FetchMarkets(ctx context.Context) ([]domain.Market, error) {
	var (
		wg      sync.WaitGroup
		spot    OKXSpotMarketResponse
		futures []OKXFutureMarket
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
		return nil, fmt.Errorf("fetch futures instruments: %w", futErr)
	}

	markets, err := mapMarkets(spot.Data, futures)
	if err != nil {
		return nil, fmt.Errorf("map markets: %w", err)
	}
	c.logger.Info("OKX FetchMarkets", slog.Int("count", len(markets)))
	return markets, nil
}

func (c *Client) fetchSpotMarket(ctx context.Context) (OKXSpotMarketResponse, error) {
	resp, err := c.httpClient.Get(ctx, spotMarketsEndpoint)
	if err != nil {
		return OKXSpotMarketResponse{}, err
	}
	defer resp.Body.Close()

	var response OKXSpotMarketResponse
	if err := decodeJSON(resp, &response); err != nil {
		return OKXSpotMarketResponse{}, err
	}
	if response.Code != "0" {
		return OKXSpotMarketResponse{}, fmt.Errorf("OKX API error: code=%s, message=%s", response.Code, response.Message)
	}
	return response, nil
}

func (c *Client) fetchFuturesMarket(ctx context.Context) ([]OKXFutureMarket, error) {
	resp, err := c.httpClient.Get(ctx, futuresMarketsEndpoint)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var response OKXFutureMarketResponse
	if err := decodeJSON(resp, &response); err != nil {
		return nil, err
	}
	if response.Code != "0" {
		return nil, fmt.Errorf("OKX API error: code=%s, message=%s", response.Code, response.Message)
	}

	futures := make([]OKXFutureMarket, 0, len(response.Data))
	for _, future := range response.Data {
		pair := strings.ToUpper(future.InstID)
		if future.State != "live" || future.SettleCcy != "USDT" || !strings.HasSuffix(pair, "-USDT-SWAP") {
			continue
		}
		futures = append(futures, future)
	}

	sem := make(chan struct{}, 4)
	var wg sync.WaitGroup
	for i := range futures {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			select {
			case sem <- struct{}{}:
			case <-ctx.Done():
				return
			}
			defer func() { <-sem }()

			endpoint := fmt.Sprintf(fundingEndpointFormat, url.QueryEscape(futures[i].InstID))
			fundingResp, err := c.httpClient.Get(ctx, endpoint)
			if err != nil {
				return
			}
			defer fundingResp.Body.Close()

			var funding OKXFutureMarketResponse
			if decodeJSON(fundingResp, &funding) != nil || funding.Code != "0" || len(funding.Data) == 0 {
				return
			}
			futures[i].FundingRate = funding.Data[0].FundingRate
			futures[i].NextFundingTime = funding.Data[0].NextFundingTime
		}(i)
	}
	wg.Wait()
	return futures, nil
}

func mapMarkets(spot []OKXSpotMarket, futures []OKXFutureMarket) ([]domain.Market, error) {
	marketsByCoin := make(map[string]*domain.Market)
	for _, symbol := range spot {
		if symbol.State != "live" || strings.ToUpper(symbol.QuoteCcy) != "USDT" {
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

	for _, future := range futures {
		pair := strings.ToUpper(future.InstID)
		if future.State != "live" || future.SettleCcy != "USDT" || !strings.HasSuffix(pair, "-USDT-SWAP") {
			continue
		}
		coin := strings.TrimSuffix(pair, "-USDT-SWAP")
		if coin == "" {
			continue
		}
		market := marketsByCoin[coin]
		if market == nil {
			market = &domain.Market{Coin: coin}
			marketsByCoin[coin] = market
		}
		market.Futurable = true
		if future.FundingRate != "" {
			fundingRate, err := strconv.ParseFloat(future.FundingRate, 64)
			if err != nil {
				return nil, fmt.Errorf("parse funding rate for %s: %w", future.InstID, err)
			}
			market.FuturesFundingRate = fundingRate
		}
		if future.NextFundingTime != "" {
			nextFundingTime, err := strconv.ParseInt(future.NextFundingTime, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("parse next funding time for %s: %w", future.InstID, err)
			}
			market.FuturesNextFundingTime = nextFundingTime
		}
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
