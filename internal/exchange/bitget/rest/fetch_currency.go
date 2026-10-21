package bitget_rest

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"github.com/MIZHT/inter-exchange-arbitrage/internal/core/domain"
)

const currenciesEndpoint = "https://api.bitget.com/api/v2/spot/public/coins"

func (c *Client) FetchCurrencies(ctx context.Context) ([]domain.Currency, error) {
	resp, err := c.httpClient.Get(ctx, currenciesEndpoint)
	if err != nil {
		return nil, fmt.Errorf("request currencies: %w", err)
	}
	defer resp.Body.Close()

	var response BitgetCurrencyResponse
	if err := decodeJSON(resp, &response); err != nil {
		return nil, fmt.Errorf("decode currencies: %w", err)
	}
	if response.Code != "00000" {
		return nil, fmt.Errorf("Bitget API error: code=%s, message=%s", response.Code, response.Msg)
	}

	currencies, err := mapCurrencies(response.Data)
	if err != nil {
		return nil, err
	}
	c.logger.Info("Bitget FetchCurrencies", slog.Int("count", len(currencies)))
	return currencies, nil
}

func mapCurrencies(assets []BitgetCurrency) ([]domain.Currency, error) {
	currencies := make([]domain.Currency, 0, len(assets))
	for _, asset := range assets {
		currency := domain.Currency{Name: strings.ToUpper(asset.Coin)}
		for _, network := range asset.Chains {
			depositEnabled := strings.EqualFold(network.Rechargeable, "true")
			withdrawEnabled := strings.EqualFold(network.Withdrawable, "true")
			if network.ContractAddress == "" || (!depositEnabled && !withdrawEnabled) {
				continue
			}
			var fee float64
			if withdrawEnabled {
				parsedFee, err := strconv.ParseFloat(network.WithdrawFee, 64)
				if err != nil {
					return nil, fmt.Errorf("parse withdraw fee for %s %s: %w", asset.Coin, network.Chain, err)
				}
				fee = parsedFee
			}
			currency.Networks = append(currency.Networks, domain.NetworkInfo{
				Name:            strings.ToUpper(network.Chain),
				ContractAddress: strings.ToUpper(network.ContractAddress),
				DepositEnabled:  depositEnabled,
				WithdrawEnabled: withdrawEnabled,
				WithdrawFee:     fee,
			})
		}
		if len(currency.Networks) > 0 {
			currencies = append(currencies, currency)
		}
	}
	return currencies, nil
}
