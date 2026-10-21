package coinex_rest

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
)

import "github.com/MIZHT/inter-exchange-arbitrage/internal/core/domain"

const currenciesEndpoint = "https://api.coinex.com/v2/assets/all-deposit-withdraw-config"

func (c *Client) FetchCurrencies(ctx context.Context) ([]domain.Currency, error) {
	resp, err := c.httpClient.Get(ctx, currenciesEndpoint)
	if err != nil {
		return nil, fmt.Errorf("request currencies: %w", err)
	}
	defer resp.Body.Close()

	var response CoinExCurrencyResponse
	if err := decodeJSON(resp, &response); err != nil {
		return nil, fmt.Errorf("decode currencies: %w", err)
	}
	if response.Code != 0 {
		return nil, fmt.Errorf("CoinEx API error: code=%d, message=%s", response.Code, response.Message)
	}

	currencies, err := mapCurrencies(response.Data)
	if err != nil {
		return nil, fmt.Errorf("map currencies: %w", err)
	}
	c.logger.Info("CoinEx FetchCurrencies", slog.Int("count", len(currencies)))
	return currencies, nil
}

func mapCurrencies(assets []CoinExCurrency) ([]domain.Currency, error) {
	currencies := make([]domain.Currency, 0, len(assets))
	for _, asset := range assets {
		currency := domain.Currency{Name: strings.ToUpper(asset.Asset.Ccy)}
		for _, network := range asset.Chains {
			contractAddress := ""
			if network.ExplorerAssetURL != "" {
				parts := strings.Split(network.ExplorerAssetURL, "/")
				if len(parts) > 0 {
					contractAddress = strings.Split(parts[len(parts)-1], "?")[0]
					if !strings.HasPrefix(contractAddress, "0x") && len(contractAddress) < 40 {
						contractAddress = ""
					}
				}
			}
			if contractAddress == "" || (!network.DepositEnabled && !network.WithdrawEnabled) {
				continue
			}
			var fee float64
			if network.WithdrawEnabled {
				parsedFee, err := strconv.ParseFloat(network.WithdrawFee, 64)
				if err != nil {
					return nil, fmt.Errorf("parse withdraw fee for %s %s: %w", asset.Asset.Ccy, network.Chain, err)
				}
				fee = parsedFee
			}
			currency.Networks = append(currency.Networks, domain.NetworkInfo{
				Name:            strings.ToUpper(network.Chain),
				ContractAddress: strings.ToUpper(contractAddress),
				DepositEnabled:  network.DepositEnabled,
				WithdrawEnabled: network.WithdrawEnabled,
				WithdrawFee:     fee,
			})
		}
		if len(currency.Networks) > 0 {
			currencies = append(currencies, currency)
		}
	}
	return currencies, nil
}
