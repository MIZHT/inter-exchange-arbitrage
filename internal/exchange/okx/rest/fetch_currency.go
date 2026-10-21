package okx_rest

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
)

import "github.com/MIZHT/inter-exchange-arbitrage/internal/core/domain"

const currenciesEndpoint = "https://www.okx.com/api/v5/asset/currencies"

func (c *Client) FetchCurrencies(ctx context.Context) ([]domain.Currency, error) {
	resp, err := c.signedGet(ctx, currenciesEndpoint)
	if err != nil {
		return nil, fmt.Errorf("request currencies: %w", err)
	}
	defer resp.Body.Close()

	var response OKXCurrencyResponse
	if err := decodeJSON(resp, &response); err != nil {
		return nil, fmt.Errorf("decode currencies: %w", err)
	}
	if response.Code != "0" {
		return nil, fmt.Errorf("OKX API error: code=%s", response.Code)
	}

	currencies, err := mapCurrencies(response.Data)
	if err != nil {
		return nil, fmt.Errorf("map currencies: %w", err)
	}
	c.logger.Info("OKX FetchCurrencies", slog.Int("count", len(currencies)))
	return currencies, nil
}

func mapCurrencies(networks []OKXCurrency) ([]domain.Currency, error) {
	currencies := make([]domain.Currency, 0, len(networks))
	currencyIndexes := make(map[string]int, len(networks))
	for _, network := range networks {
		if network.ContractAddress == "" || (!network.DepositEnabled && !network.WithdrawEnabled) {
			continue
		}
		var fee float64
		if network.WithdrawEnabled {
			parsedFee, err := strconv.ParseFloat(network.WithdrawFee, 64)
			if err != nil {
				return nil, fmt.Errorf("parse withdraw fee for %s %s: %w", network.Currency, network.Chain, err)
			}
			fee = parsedFee
		}
		name := strings.ToUpper(network.Currency)
		index, exists := currencyIndexes[name]
		if !exists {
			index = len(currencies)
			currencyIndexes[name] = index
			currencies = append(currencies, domain.Currency{Name: name})
		}
		currencies[index].Networks = append(currencies[index].Networks, domain.NetworkInfo{
			Name:            strings.ToUpper(network.Chain),
			ContractAddress: strings.ToUpper(network.ContractAddress),
			DepositEnabled:  network.DepositEnabled,
			WithdrawEnabled: network.WithdrawEnabled,
			WithdrawFee:     fee,
		})
	}
	return currencies, nil
}
