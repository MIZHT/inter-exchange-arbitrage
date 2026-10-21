package mexc_rest

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"strconv"
	"strings"

	"github.com/MIZHT/inter-exchange-arbitrage/internal/core/domain"
)

const currenciesEndpoint = "https://api.mexc.com/api/v3/capital/config/getall"

func (c *Client) FetchCurrencies(ctx context.Context) ([]domain.Currency, error) {
	resp, err := c.signedGet(ctx, currenciesEndpoint, url.Values{})
	if err != nil {
		return nil, fmt.Errorf("request currencies: %w", err)
	}
	defer resp.Body.Close()

	var response []MEXCCurrency
	if err := decodeJSON(resp, &response); err != nil {
		return nil, fmt.Errorf("decode currencies: %w", err)
	}

	currencies, err := mapCurrencies(response)
	if err != nil {
		return nil, fmt.Errorf("map currencies: %w", err)
	}
	c.logger.Info("MEXC FetchCurrencies", slog.Int("count", len(currencies)))
	return currencies, nil
}

func mapCurrencies(coins []MEXCCurrency) ([]domain.Currency, error) {
	currencies := make([]domain.Currency, 0, len(coins))
	for _, coin := range coins {
		currency := domain.Currency{Name: strings.ToUpper(coin.Coin)}
		for _, network := range coin.NetworkList {
			if network.ContractAddress == "" || (!network.DepositEnable && !network.WithdrawEnable) {
				continue
			}
			var fee float64
			if network.WithdrawEnable {
				parsedFee, err := strconv.ParseFloat(network.WithdrawFee, 64)
				if err != nil {
					return nil, fmt.Errorf("parse withdraw fee for %s %s: %w", coin.Coin, network.Network, err)
				}
				fee = parsedFee
			}
			currency.Networks = append(currency.Networks, domain.NetworkInfo{
				Name:            strings.ToUpper(network.Network),
				ContractAddress: strings.ToUpper(network.ContractAddress),
				DepositEnabled:  network.DepositEnable,
				WithdrawEnabled: network.WithdrawEnable,
				WithdrawFee:     fee,
			})
		}
		if len(currency.Networks) > 0 {
			currencies = append(currencies, currency)
		}
	}
	return currencies, nil
}
