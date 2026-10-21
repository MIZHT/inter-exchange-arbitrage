package binance

import (
	"context"
	"fmt"

	core_config "github.com/MIZHT/inter-exchange-arbitrage/internal/core/config"
	"github.com/MIZHT/inter-exchange-arbitrage/internal/core/domain"
	core_logger "github.com/MIZHT/inter-exchange-arbitrage/internal/core/logger"
	"github.com/MIZHT/inter-exchange-arbitrage/internal/core/transport/http_client"
	binance_rest "github.com/MIZHT/inter-exchange-arbitrage/internal/exchange/binance/rest"
)

type Binance struct {
	log  *core_logger.Logger
	rest *binance_rest.Client
	//wsManager  *websocket.Manager
	exCfg core_config.Exchange
}

func NewBinance(
	log *core_logger.Logger,
	exCfg core_config.Exchange,
	proxy string,
	//wsManager *websocket.Manager,
) *Binance {
	httpClient := http_client.NewHTTPClient(
		http_client.WithLogger(log.Logger),
	)

	return &Binance{
		log:   log,
		exCfg: exCfg,
		rest:  binance_rest.NewClient(httpClient, log, exCfg),
		//wsManager:  wsManager,
	}
}

func (ex *Binance) FetchCurrencies(ctx context.Context) ([]domain.Currency, error) {
	currencies, err := ex.rest.FetchCurrencies(ctx)
	if err != nil {
		return nil, fmt.Errorf("binance fetch currencies: %w", err)
	}
	return currencies, nil
}

func (ex *Binance) FetchMarkets(ctx context.Context) ([]domain.Market, error) {
	markets, err := ex.rest.FetchMarkets(ctx)
	if err != nil {
		return nil, fmt.Errorf("binance fetch markets: %w", err)
	}
	return markets, nil
}

// func (ex *Binance) SubscribeOrderBook(ctx context.Context, symbol string) error {
// 	err := binance_websocket.Subscribe(ctx, symbol)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return nil
// }

// func (ex *Binance) UnsubscribeOrderBook(ctx context.Context, symbol string) error {
// 	err := binance_websocket.Unsubscribe(ctx, symbol)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return nil
// }
