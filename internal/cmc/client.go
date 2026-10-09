package cmc

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/MIZHT/inter-exchange-arbitrage/internal/core/transport/http_client"
)

const (
	indexURL       = "https://s3.coinmarketcap.com/generated/core/crypto/web-search.json"
	marketPairsURL = "https://api.coinmarketcap.com/data-api/v3/cryptocurrency/market-pairs/latest"
	usdtQuoteID    = "825"
	pageLimit      = 200
)

func FetchIndex(ctx context.Context, client *http_client.Client) ([]int, error) {
	resp, err := client.Get(ctx, indexURL)
	if err != nil {
		return nil, fmt.Errorf("fetch CMC index: %w", err)
	}
	defer resp.Body.Close()

	var index CmcIndexResponse
	if err := decodeJSON(resp, &index); err != nil {
		return nil, fmt.Errorf("decode CMC index: %w", err)
	}

	symbols, err := listIndex(index)
	if err != nil {
		return nil, fmt.Errorf("map CMC index: %w", err)
	}
	return symbols, nil
}

func FetchMarketPairs(
	ctx context.Context,
	client *http_client.Client,
	coinID int,
) ([]MarketPair, string, error) {

	reqURL := marketPairsRequestURL(coinID, pageLimit)
	resp, err := client.Get(ctx, reqURL)
	if err != nil {
		return nil, "", fmt.Errorf("fetch CMC market pairs for coin %d: %w", coinID, err)
	}

	var apiResp MarketPairsResponse
	if err := decodeAndClose(resp, &apiResp); err != nil {
		return nil, "", fmt.Errorf("decode CMC market pairs for coin %d: %w", coinID, err)
	}
	if len(apiResp.Data.MarketPairs) != 0 {
		CoinName := apiResp.Data.MarketPairs[0].BaseSymbol

		markets := make([]MarketPair, 0, len(apiResp.Data.MarketPairs))
		for _, market := range apiResp.Data.MarketPairs {
			markets = append(markets, market)
		}
		return markets, CoinName, nil
	}
	return nil, "", fmt.Errorf("cmc marketsPairs request is empty")
}

func marketPairsRequestURL(coinID, limit int) string {
	q := url.Values{}
	q.Set("id", strconv.Itoa(coinID))
	q.Set("category", "all")
	q.Set("limit", strconv.Itoa(limit))
	q.Set("quoteCurrencyId", usdtQuoteID)
	return marketPairsURL + "?" + q.Encode()
}

func decodeAndClose(resp *http.Response, dst any) error {
	defer resp.Body.Close()
	return decodeJSON(resp, dst)
}

func decodeJSON(resp *http.Response, dst any) error {
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusBadRequest {
		return fmt.Errorf("unexpected status %d", resp.StatusCode)
	}
	if err := json.NewDecoder(resp.Body).Decode(dst); err != nil {
		return fmt.Errorf("decode json: %w", err)
	}
	return nil
}
