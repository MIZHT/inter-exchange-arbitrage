package bybit_rest

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"strconv"
	"strings"

	"github.com/MIZHT/inter-exchange-arbitrage/internal/core/domain"
)

const currenciesEndpoint = "https://api.bybit.com/v5/asset/coin/query-info"

func (c *Client) FetchCurrencies(ctx context.Context) ([]domain.Currency, error) {
	resp, err := c.signedGet(ctx, currenciesEndpoint, url.Values{})
	if err != nil {
		return nil, fmt.Errorf("request currency config: %w", err)
	}
	defer resp.Body.Close()

	var response BybitCurrencyResponse
	if err := decodeJSON(resp, &response); err != nil {
		return nil, fmt.Errorf("decode currency config: %w", err)
	}
	if response.RetCode != 0 {
		return nil, fmt.Errorf("Bybit API error: retCode=%d, retMsg=%s", response.RetCode, response.RetMsg)
	}

	currencies, err := mapCurrencies(response.Result.Rows)
	if err != nil {
		return nil, fmt.Errorf("map currencies: %w", err)
	}
	c.logger.Info("Bybit FetchCurrencies", slog.Int("count", len(currencies)))
	return currencies, nil
}

func mapCurrencies(coins []BybitCurrency) ([]domain.Currency, error) {
	currencies := make([]domain.Currency, 0, len(coins))
	for _, coin := range coins {
		currency := domain.Currency{Name: strings.ToUpper(coin.Coin)}
		for _, network := range coin.Chains {
			if network.ContractAddress == "" || (network.DepositDisabled && network.WithdrawDisabled) {
				continue
			}
			var fee float64
			if !network.WithdrawDisabled {
				parsedFee, err := strconv.ParseFloat(network.WithdrawFee, 64)
				if err != nil {
					return nil, fmt.Errorf("parse withdraw fee for %s %s: %w", coin.Coin, network.Chain, err)
				}
				fee = parsedFee
			}
			currency.Networks = append(currency.Networks, domain.NetworkInfo{
				Name:            strings.ToUpper(network.Chain),
				ContractAddress: strings.ToUpper(network.ContractAddress),
				DepositEnabled:  !network.DepositDisabled,
				WithdrawEnabled: !network.WithdrawDisabled,
				WithdrawFee:     fee,
			})
		}
		if len(currency.Networks) > 0 {
			currencies = append(currencies, currency)
		}
	}
	return currencies, nil
}
