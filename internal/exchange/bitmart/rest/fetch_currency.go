package bitmart_rest

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"github.com/MIZHT/inter-exchange-arbitrage/internal/core/domain"
)

const currenciesEndpoint = "https://api-cloud.bitmart.com/account/v1/currencies"

func (c *Client) FetchCurrencies(ctx context.Context) ([]domain.Currency, error) {
	resp, err := c.httpClient.Get(ctx, currenciesEndpoint)
	if err != nil {
		return nil, fmt.Errorf("request currencies: %w", err)
	}
	defer resp.Body.Close()

	var response BitmartCurrencyResponse
	if err := decodeJSON(resp, &response); err != nil {
		return nil, fmt.Errorf("decode currencies: %w", err)
	}
	code := fmt.Sprint(response.Code)
	if code != "1000" && code != "0" {
		return nil, fmt.Errorf("BitMart API error: code=%s, message=%s", code, response.Message)
	}

	currencies, err := mapCurrencies(response.Data.Currencies)
	if err != nil {
		return nil, err
	}
	c.logger.Info("BitMart FetchCurrencies", slog.Int("count", len(currencies)))
	return currencies, nil
}

func mapCurrencies(assets []BitmartCurrency) ([]domain.Currency, error) {
	currencies := make([]domain.Currency, 0, len(assets))
	currencyIndexes := make(map[string]int, len(assets))
	for _, asset := range assets {
		contractAddress := ""
		if asset.ContractAddress != nil {
			contractAddress = *asset.ContractAddress
		}
		if contractAddress == "" || (!asset.DepositEnabled && !asset.WithdrawEnabled) {
			continue
		}
		var fee float64
		if asset.WithdrawEnabled {
			parsedFee, err := strconv.ParseFloat(asset.WithdrawFee, 64)
			if err != nil {
				return nil, fmt.Errorf("parse withdraw fee for %s %s: %w", asset.Currency, asset.Network, err)
			}
			fee = parsedFee
		}
		name := strings.ToUpper(asset.Currency)
		index, exists := currencyIndexes[name]
		if !exists {
			index = len(currencies)
			currencyIndexes[name] = index
			currencies = append(currencies, domain.Currency{Name: name})
		}
		currencies[index].Networks = append(currencies[index].Networks, domain.NetworkInfo{
			Name:            strings.ToUpper(asset.Network),
			ContractAddress: strings.ToUpper(contractAddress),
			DepositEnabled:  asset.DepositEnabled,
			WithdrawEnabled: asset.WithdrawEnabled,
			WithdrawFee:     fee,
		})
	}
	return currencies, nil
}
