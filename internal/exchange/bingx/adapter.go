package bingx

import (
	"context"
	"fmt"

	core_config "github.com/MIZHT/inter-exchange-arbitrage/internal/core/config"
	"github.com/MIZHT/inter-exchange-arbitrage/internal/core/domain"
	core_logger "github.com/MIZHT/inter-exchange-arbitrage/internal/core/logger"
	"github.com/MIZHT/inter-exchange-arbitrage/internal/core/transport/http_client"
	bingx_rest "github.com/MIZHT/inter-exchange-arbitrage/internal/exchange/bingx/rest"
)

type Bingx struct {
	log   *core_logger.Logger
	rest  *bingx_rest.Client
	exCfg core_config.Exchange
}

func NewBingx(
	log *core_logger.Logger,
	exCfg core_config.Exchange,
	proxy string,
) *Bingx {
	httpClient := http_client.NewHTTPClient(
		http_client.WithLogger(log.Logger),
	)

	return &Bingx{
		log:   log,
		exCfg: exCfg,
		rest:  bingx_rest.NewClient(httpClient, log, exCfg),
	}
}

func (ex *Bingx) FetchCurrencies(ctx context.Context) ([]domain.Currency, error) {
	currencies, err := ex.rest.FetchCurrencies(ctx)
	if err != nil {
		return nil, fmt.Errorf("bingx fetch currencies: %w", err)
	}
	return currencies, nil
}

func (ex *Bingx) FetchMarkets(ctx context.Context) ([]domain.Market, error) {
	markets, err := ex.rest.FetchMarkets(ctx)
	if err != nil {
		return nil, fmt.Errorf("bingx fetch markets: %w", err)
	}
	return markets, nil
}
