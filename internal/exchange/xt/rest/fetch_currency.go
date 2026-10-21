package xt_rest

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"

	"github.com/MIZHT/inter-exchange-arbitrage/internal/core/domain"
)

const (
	currenciesEndpoint = baseURL + "/v4/public/currencies"
	chainsEndpoint     = baseURL + "/v4/public/wallet/support/currency"
)

func (c *Client) FetchCurrencies(ctx context.Context) ([]domain.Currency, error) {
	var (
		wg           sync.WaitGroup
		currencies   XTCurrencyResponse
		chains       XTNetworkResponse
		currencyErr  error
		chainInfoErr error
	)
	wg.Add(2)
	go func() {
		defer wg.Done()
		resp, err := c.httpClient.Get(ctx, currenciesEndpoint)
		if err != nil {
			currencyErr = err
			return
		}
		defer resp.Body.Close()
		currencyErr = decodeJSON(resp, &currencies)
	}()
	go func() {
		defer wg.Done()
		resp, err := c.httpClient.Get(ctx, chainsEndpoint)
		if err != nil {
			chainInfoErr = err
			return
		}
		defer resp.Body.Close()
		chainInfoErr = decodeJSON(resp, &chains)
	}()
	wg.Wait()
	if currencyErr != nil {
		return nil, fmt.Errorf("request or decode currencies: %w", currencyErr)
	}
	if chainInfoErr != nil {
		return nil, fmt.Errorf("request or decode network configuration: %w", chainInfoErr)
	}
	if currencies.ReturnCode != 0 {
		return nil, fmt.Errorf("XT currencies API error: %s (code %d)", currencies.Message, currencies.ReturnCode)
	}
	if chains.ReturnCode != 0 {
		return nil, fmt.Errorf("XT network configuration API error: %s (code %d)", chains.Message, chains.ReturnCode)
	}

	result, err := mapCurrencies(currencies.Result.Currencies, chains.Result)
	if err != nil {
		return nil, fmt.Errorf("map currencies: %w", err)
	}
	c.logger.Info("XT FetchCurrencies", slog.Int("count", len(result)))
	return result, nil
}

func mapCurrencies(coins []XTCurrency, currencyNetworks []XTNetworkCurrency) ([]domain.Currency, error) {
	networksByCurrency := make(map[string][]XTNetwork, len(currencyNetworks))
	for _, entry := range currencyNetworks {
		networksByCurrency[strings.ToUpper(entry.Currency)] = entry.SupportChains
	}

	currencies := make([]domain.Currency, 0, len(coins))
	for _, coin := range coins {
		if coin.IsListing != 1 {
			continue
		}
		name := strings.ToUpper(coin.Currency)
		currency := domain.Currency{Name: name}
		for _, network := range networksByCurrency[name] {
			depositEnabled := network.DepositEnabled && coin.DepositStatus == 1
			withdrawEnabled := network.WithdrawEnabled && coin.WithdrawStatus == 1
			if network.ContractAddress == "" || (!depositEnabled && !withdrawEnabled) {
				continue
			}
			currency.Networks = append(currency.Networks, domain.NetworkInfo{
				Name:            strings.ToUpper(network.Chain),
				ContractAddress: strings.ToUpper(network.ContractAddress),
				DepositEnabled:  depositEnabled,
				WithdrawEnabled: withdrawEnabled,
				WithdrawFee:     network.WithdrawFee,
			})
		}
		if len(currency.Networks) > 0 {
			currencies = append(currencies, currency)
		}
	}
	return currencies, nil
}
