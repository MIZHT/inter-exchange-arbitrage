package bybit

import (
	"context"

	core_config "github.com/MIZHT/inter-exchange-arbitrage/internal/core/config"
	"github.com/MIZHT/inter-exchange-arbitrage/internal/core/domain"
	core_logger "github.com/MIZHT/inter-exchange-arbitrage/internal/core/logger"
	"github.com/MIZHT/inter-exchange-arbitrage/internal/core/transport/http_client"
	bybit_rest "github.com/MIZHT/inter-exchange-arbitrage/internal/exchange/bybit/rest"
)

type Bybit struct {
	log        *core_logger.Logger
	restClient *http_client.Client
	//wsManager  *websocket.Manager
	exCfg core_config.Exchange
}

func NewBybit(
	log *core_logger.Logger,
	exCfg core_config.Exchange,
	//wsManager *websocket.Manager,
) *Bybit {
	return &Bybit{
		log:   log,
		exCfg: exCfg,
		restClient: http_client.NewHTTPClient(
			http_client.WithLogger(log.Logger),
		),
		//wsManager:  wsManager,
	}
}

func (ex *Bybit) Name() string {
	return string(core_config.Bybit)
}

func (ex *Bybit) FetchCurrencies(ctx context.Context) ([]domain.Currency, error) {
	currencies, err := bybit_rest.FetchCurrencies(ctx)
	if err != nil {
		return nil, err
	}
	return currencies, err
}

func (ex *Bybit) FetchMarkets(ctx context.Context) ([]domain.Market, error) {
	markets, err := bybit_rest.FetchMarkets(ctx)
	if err != nil {
		return nil, err
	}
	return markets, err
}

// func (ex *Bybit) SubscribeOrderBook(ctx context.Context, symbol string) error {
// 	err := bybit_websocket.Subscribe(ctx, symbol)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return nil
// }

// func (ex *Bybit) UnsubscribeOrderBook(ctx context.Context, symbol string) error {
// 	err := bybit_websocket.Unsubscribe(ctx, symbol)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return nil
// }
