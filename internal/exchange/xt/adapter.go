package xt

import (
	"context"
	"fmt"

	core_config "github.com/MIZHT/inter-exchange-arbitrage/internal/core/config"
	"github.com/MIZHT/inter-exchange-arbitrage/internal/core/domain"
	core_logger "github.com/MIZHT/inter-exchange-arbitrage/internal/core/logger"
	"github.com/MIZHT/inter-exchange-arbitrage/internal/core/transport/http_client"
	rest "github.com/MIZHT/inter-exchange-arbitrage/internal/exchange/xt/rest"
)

type XT struct {
	log   *core_logger.Logger
	rest  *rest.Client
	exCfg core_config.Exchange
}

func NewXT(log *core_logger.Logger, exCfg core_config.Exchange, proxy string) *XT {
	hc := http_client.NewHTTPClient(http_client.WithLogger(log.Logger), http_client.WithProxy(proxy))
	return &XT{log: log, exCfg: exCfg, rest: rest.NewClient(hc, log, exCfg)}
}
func (ex *XT) FetchCurrencies(ctx context.Context) ([]domain.Currency, error) {
	v, err := ex.rest.FetchCurrencies(ctx)
	if err != nil {
		return nil, fmt.Errorf("xt fetch currencies: %w", err)
	}
	return v, nil
}
func (ex *XT) FetchMarkets(ctx context.Context) ([]domain.Market, error) {
	v, err := ex.rest.FetchMarkets(ctx)
	if err != nil {
		return nil, fmt.Errorf("xt fetch markets: %w", err)
	}
	return v, nil
}
