package xt

import (
	"context"
	"fmt"

	core_config "github.com/MIZHT/inter-exchange-arbitrage/internal/core/config"
	"github.com/MIZHT/inter-exchange-arbitrage/internal/core/domain"
	core_logger "github.com/MIZHT/inter-exchange-arbitrage/internal/core/logger"
	"github.com/MIZHT/inter-exchange-arbitrage/internal/core/transport/http_client"
	"github.com/MIZHT/inter-exchange-arbitrage/internal/core/transport/websocket_manager"
	"github.com/MIZHT/inter-exchange-arbitrage/internal/exchange"
	rest "github.com/MIZHT/inter-exchange-arbitrage/internal/exchange/xt/rest"
	xt_ws "github.com/MIZHT/inter-exchange-arbitrage/internal/exchange/xt/websocket"
)

type XT struct {
	log   *core_logger.Logger
	rest  *rest.Client
	ws    *xt_ws.Manager
	exCfg core_config.Exchange
}

func NewXT(log *core_logger.Logger, exCfg core_config.Exchange, proxy string, pools ...*websocket_manager.ProxyPool) *XT {
	hc := http_client.NewHTTPClient(http_client.WithLogger(log.Logger), http_client.WithProxy(proxy))
	return &XT{log: log, exCfg: exCfg, rest: rest.NewClient(hc, log, exCfg), ws: xt_ws.NewManager(proxy, pools...)}
}

func (ex *XT) StartOrderBookStream(ctx context.Context, handlers exchange.OrderBookHandlers) error {
	return ex.ws.Start(ctx, func(market websocket_manager.Market, book domain.OrderBook) {
		if market == websocket_manager.Spot && handlers.Spot != nil {
			handlers.Spot(book)
		} else if market == websocket_manager.Futures && handlers.Futures != nil {
			handlers.Futures(book)
		}
	})
}
func (ex *XT) SubscribeSpotOrderBooks(ctx context.Context, coins []string) error {
	return ex.ws.SubscribeSpot(ctx, coins)
}
func (ex *XT) UnsubscribeSpotOrderBooks(ctx context.Context, coins []string) error {
	return ex.ws.UnsubscribeSpot(ctx, coins)
}
func (ex *XT) SubscribeFuturesOrderBooks(ctx context.Context, coins []string) error {
	return ex.ws.SubscribeFutures(ctx, coins)
}
func (ex *XT) UnsubscribeFuturesOrderBooks(ctx context.Context, coins []string) error {
	return ex.ws.UnsubscribeFutures(ctx, coins)
}
func (ex *XT) ReconcileOrderBooks(ctx context.Context, spot, futures []string) error {
	return ex.ws.ReconcileOrderBooks(ctx, spot, futures)
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
