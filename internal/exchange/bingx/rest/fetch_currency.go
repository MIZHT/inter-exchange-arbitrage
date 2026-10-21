package bingx_rest

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/MIZHT/inter-exchange-arbitrage/internal/core/domain"
)

func (c *Client) FetchCurrencies(ctx context.Context) ([]domain.Currency, error) {
	resp, err := c.signedGet(ctx, currenciesEndpoint, url.Values{})
	if err != nil {
		return nil, fmt.Errorf("request currency config: %w", err)
	}
	defer resp.Body.Close()

	var response BingXCurrencyResponse
	if err := decodeJSON(resp, &response); err != nil {
		return nil, fmt.Errorf("decode currency config: %w", err)
	}
	if response.Code != 0 {
		return nil, fmt.Errorf("BingX API error: code=%d, message=%s", response.Code, response.Msg)
	}
	return mapCurrencies(response)
}

func mapCurrencies(response BingXCurrencyResponse) ([]domain.Currency, error) {
	currencies := make([]domain.Currency, 0, len(response.Data))
	for _, asset := range response.Data {
		currency := domain.Currency{Name: strings.ToUpper(asset.Coin)}
		for _, network := range asset.NetworkList {
			if network.ContractAddress == "" || (!network.DepositEnable && !network.WithdrawEnable) {
				continue
			}
			var fee float64
			if network.WithdrawEnable {
				parsedFee, err := strconv.ParseFloat(network.WithdrawFee, 64)
				if err != nil {
					return nil, fmt.Errorf("parse withdraw fee for %s %s: %w", asset.Coin, network.Network, err)
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
