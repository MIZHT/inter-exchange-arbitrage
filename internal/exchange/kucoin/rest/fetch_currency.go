package kucoin_rest

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"github.com/MIZHT/inter-exchange-arbitrage/internal/core/domain"
)

const currenciesEndpoint = "https://api.kucoin.com/api/v1/currencies"

func (c *Client) FetchCurrencies(ctx context.Context) ([]domain.Currency, error) {
	resp, err := c.httpClient.Get(ctx, currenciesEndpoint)
	if err != nil {
		return nil, fmt.Errorf("request currencies: %w", err)
	}
	defer resp.Body.Close()

	var response KuCoinCurrencyResponse
	if err := decodeJSON(resp, &response); err != nil {
		return nil, fmt.Errorf("decode currencies: %w", err)
	}
	if response.Code != "200000" {
		return nil, fmt.Errorf("KuCoin API error: code=%s", response.Code)
	}

	currencies, err := mapCurrencies(response.Data)
	if err != nil {
		return nil, fmt.Errorf("map currencies: %w", err)
	}
	c.logger.Info("KuCoin FetchCurrencies", slog.Int("count", len(currencies)))
	return currencies, nil
}

func mapCurrencies(coins []KuCoinCurrency) ([]domain.Currency, error) {
	currencies := make([]domain.Currency, 0, len(coins))
	for _, coin := range coins {
		if coin.ContractAddress == "" || (!coin.IsDepositEnabled && !coin.IsWithdrawEnabled) {
			continue
		}
		var fee float64
		if coin.IsWithdrawEnabled {
			parsedFee, err := strconv.ParseFloat(coin.WithdrawalMinFee, 64)
			if err != nil {
				return nil, fmt.Errorf("parse withdraw fee for %s: %w", coin.Currency, err)
			}
			fee = parsedFee
		}
		currency := domain.Currency{
			Name: strings.ToUpper(coin.Currency),
			Networks: []domain.NetworkInfo{{
				Name:            strings.ToUpper(coin.Currency),
				ContractAddress: strings.ToUpper(coin.ContractAddress),
				DepositEnabled:  coin.IsDepositEnabled,
				WithdrawEnabled: coin.IsWithdrawEnabled,
				WithdrawFee:     fee,
			}},
		}
		currencies = append(currencies, currency)
	}
	return currencies, nil
}
