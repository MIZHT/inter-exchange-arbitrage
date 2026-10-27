package coinex

import (
	"context"
	"fmt"

	core_config "github.com/MIZHT/inter-exchange-arbitrage/internal/core/config"
	"github.com/MIZHT/inter-exchange-arbitrage/internal/core/domain"
	core_logger "github.com/MIZHT/inter-exchange-arbitrage/internal/core/logger"
	"github.com/MIZHT/inter-exchange-arbitrage/internal/core/transport/http_client"
	"github.com/MIZHT/inter-exchange-arbitrage/internal/core/transport/websocket_manager"
	"github.com/MIZHT/inter-exchange-arbitrage/internal/exchange"
	rest "github.com/MIZHT/inter-exchange-arbitrage/internal/exchange/coinex/rest"
	coinex_ws "github.com/MIZHT/inter-exchange-arbitrage/internal/exchange/coinex/websocket"
)

type CoinEx struct {
	log   *core_logger.Logger
	rest  *rest.Client
	ws    *coinex_ws.Manager
	exCfg core_config.Exchange
}

func NewCoinEx(log *core_logger.Logger, exCfg core_config.Exchange, proxy string, pools ...*websocket_manager.ProxyPool) *CoinEx {
	hc := http_client.NewHTTPClient(http_client.WithLogger(log.Logger))
	return &CoinEx{log: log, exCfg: exCfg, rest: rest.NewClient(hc, log, exCfg), ws: coinex_ws.NewManager(proxy, pools...)}
}

func (ex *CoinEx) StartOrderBookStream(ctx context.Context, handlers exchange.OrderBookHandlers) error {
	return ex.ws.Start(ctx, func(market websocket_manager.Market, book domain.OrderBook) {
		if market == websocket_manager.Spot && handlers.Spot != nil {
			handlers.Spot(book)
		} else if market == websocket_manager.Futures && handlers.Futures != nil {
			handlers.Futures(book)
		}
	})
}
func (ex *CoinEx) SubscribeSpotOrderBooks(ctx context.Context, coins []string) error {
	return ex.ws.SubscribeSpot(ctx, coins)
}
func (ex *CoinEx) UnsubscribeSpotOrderBooks(ctx context.Context, coins []string) error {
	return ex.ws.UnsubscribeSpot(ctx, coins)
}
func (ex *CoinEx) SubscribeFuturesOrderBooks(ctx context.Context, coins []string) error {
	return ex.ws.SubscribeFutures(ctx, coins)
}
func (ex *CoinEx) UnsubscribeFuturesOrderBooks(ctx context.Context, coins []string) error {
	return ex.ws.UnsubscribeFutures(ctx, coins)
}
func (ex *CoinEx) ReconcileOrderBooks(ctx context.Context, spot, futures []string) error {
	return ex.ws.ReconcileOrderBooks(ctx, spot, futures)
}
func (ex *CoinEx) FetchCurrencies(ctx context.Context) ([]domain.Currency, error) {
	v, err := ex.rest.FetchCurrencies(ctx)
	if err != nil {
		return nil, fmt.Errorf("coinex fetch currencies: %w", err)
	}
	return v, nil
}
func (ex *CoinEx) FetchMarkets(ctx context.Context) ([]domain.Market, error) {
	v, err := ex.rest.FetchMarkets(ctx)
	if err != nil {
		return nil, fmt.Errorf("coinex fetch markets: %w", err)
	}
	return v, nil
}
