package gateio_rest

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"github.com/MIZHT/inter-exchange-arbitrage/internal/core/domain"
)

const currenciesEndpoint = "https://api.gateio.ws/api/v4/spot/currencies"

func (c *Client) FetchCurrencies(ctx context.Context) ([]domain.Currency, error) {
	resp, err := c.httpClient.Get(ctx, currenciesEndpoint)
	if err != nil {
		return nil, fmt.Errorf("request currencies: %w", err)
	}
	defer resp.Body.Close()

	var response []GateIOCurrency
	if err := decodeJSON(resp, &response); err != nil {
		return nil, fmt.Errorf("decode currencies: %w", err)
	}

	currencies, err := mapCurrencies(response)
	if err != nil {
		return nil, fmt.Errorf("map currencies: %w", err)
	}
	c.logger.Info("GateIO FetchCurrencies", slog.Int("count", len(currencies)))
	return currencies, nil
}

func mapCurrencies(assets []GateIOCurrency) ([]domain.Currency, error) {
	currencies := make([]domain.Currency, 0, len(assets))
	for _, asset := range assets {
		if asset.Delisted || asset.TradeDisabled {
			continue
		}
		currency := domain.Currency{Name: strings.ToUpper(asset.Currency)}
		for _, network := range asset.Chains {
			depositEnabled := !network.DepositDisabled
			withdrawEnabled := !network.WithdrawDisabled
			if network.Address == "" || (!depositEnabled && !withdrawEnabled) {
				continue
			}
			var fee float64
			if withdrawEnabled {
				parsedFee, err := strconv.ParseFloat(network.WithdrawFee, 64)
				if err != nil {
					return nil, fmt.Errorf("parse withdraw fee for %s %s: %w", asset.Currency, network.Name, err)
				}
				fee = parsedFee
			}
			currency.Networks = append(currency.Networks, domain.NetworkInfo{
				Name:            strings.ToUpper(network.Name),
				ContractAddress: strings.ToUpper(network.Address),
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
