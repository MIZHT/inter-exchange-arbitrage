package binance_rest

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/MIZHT/inter-exchange-arbitrage/internal/core/domain"
	"go.uber.org/zap"
)

func (c *Client) FetchCurrencies(ctx context.Context) ([]domain.Currency, error) {
	resp, err := c.signedGet(ctx, currenciesEndpoint, url.Values{})
	if err != nil {
		return nil, fmt.Errorf("request asset configs: %w", err)
	}
	defer resp.Body.Close()

	var assets BinanceCurrencies
	if err := decodeJSON(resp, &assets); err != nil {
		return nil, fmt.Errorf("decode asset configs: %w", err)
	}

	currencies, err := mapCurrencies(assets)
	if err != nil {
		return nil, fmt.Errorf("map asset configs: %w", err)
	}

	c.logger.Info("Binance FetchCurrencies", zap.Int("count", len(currencies)))
	return currencies, nil
}

func mapCurrencies(assets BinanceCurrencies) ([]domain.Currency, error) {
	currencies := make([]domain.Currency, 0, len(assets))
	for _, asset := range assets {
		cur := domain.Currency{Name: strings.ToUpper(asset.Coin)}

		for _, net := range asset.NetworkList {
			if net.ContractAddress == "" {
				continue
			}
			if !net.DepositEnable || !net.WithdrawEnable {
				continue
			}

			fee, err := strconv.ParseFloat(net.WithdrawFee, 64)
			if err != nil {
				return nil, fmt.Errorf("parse withdraw fee for %s %s: %w", asset.Coin, net.Network, err)
			}

			cur.Networks = append(cur.Networks, domain.NetworkInfo{
				Name:            net.Network,
				ContractAddress: strings.ToLower(net.ContractAddress),
				DepositEnabled:  net.DepositEnable,
				WithdrawEnabled: net.WithdrawEnable,
				WithdrawFee:     fee,
			})
		}

		if len(cur.Networks) == 0 {
			continue
		}
		currencies = append(currencies, cur)
	}
	return currencies, nil
}
