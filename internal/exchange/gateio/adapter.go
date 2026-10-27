package gateio

import (
	"context"
	"fmt"

	core_config "github.com/MIZHT/inter-exchange-arbitrage/internal/core/config"
	"github.com/MIZHT/inter-exchange-arbitrage/internal/core/domain"
	core_logger "github.com/MIZHT/inter-exchange-arbitrage/internal/core/logger"
	"github.com/MIZHT/inter-exchange-arbitrage/internal/core/transport/http_client"
	"github.com/MIZHT/inter-exchange-arbitrage/internal/core/transport/websocket_manager"
	"github.com/MIZHT/inter-exchange-arbitrage/internal/exchange"
	rest "github.com/MIZHT/inter-exchange-arbitrage/internal/exchange/gateio/rest"
	gateio_ws "github.com/MIZHT/inter-exchange-arbitrage/internal/exchange/gateio/websocket"
)

type GateIO struct {
	log   *core_logger.Logger
	rest  *rest.Client
	ws    *gateio_ws.Manager
	exCfg core_config.Exchange
}

func NewGateIO(log *core_logger.Logger, exCfg core_config.Exchange, proxy string, pools ...*websocket_manager.ProxyPool) *GateIO {
	hc := http_client.NewHTTPClient(http_client.WithLogger(log.Logger), http_client.WithProxy(proxy))
	return &GateIO{log: log, exCfg: exCfg, rest: rest.NewClient(hc, log, exCfg), ws: gateio_ws.NewManager(proxy, pools...)}
}

func (ex *GateIO) StartOrderBookStream(ctx context.Context, handlers exchange.OrderBookHandlers) error {
	return ex.ws.Start(ctx, func(market websocket_manager.Market, book domain.OrderBook) {
		if market == websocket_manager.Spot && handlers.Spot != nil {
			handlers.Spot(book)
		} else if market == websocket_manager.Futures && handlers.Futures != nil {
			handlers.Futures(book)
		}
	})
}
func (ex *GateIO) SubscribeSpotOrderBooks(ctx context.Context, coins []string) error {
	return ex.ws.SubscribeSpot(ctx, coins)
}
func (ex *GateIO) UnsubscribeSpotOrderBooks(ctx context.Context, coins []string) error {
	return ex.ws.UnsubscribeSpot(ctx, coins)
}
func (ex *GateIO) SubscribeFuturesOrderBooks(ctx context.Context, coins []string) error {
	return ex.ws.SubscribeFutures(ctx, coins)
}
func (ex *GateIO) UnsubscribeFuturesOrderBooks(ctx context.Context, coins []string) error {
	return ex.ws.UnsubscribeFutures(ctx, coins)
}
func (ex *GateIO) ReconcileOrderBooks(ctx context.Context, spot, futures []string) error {
	return ex.ws.ReconcileOrderBooks(ctx, spot, futures)
}
func (ex *GateIO) FetchCurrencies(ctx context.Context) ([]domain.Currency, error) {
	v, err := ex.rest.FetchCurrencies(ctx)
	if err != nil {
		return nil, fmt.Errorf("gateio fetch currencies: %w", err)
	}
	return v, nil
}
func (ex *GateIO) FetchMarkets(ctx context.Context) ([]domain.Market, error) {
	v, err := ex.rest.FetchMarkets(ctx)
	if err != nil {
		return nil, fmt.Errorf("gateio fetch markets: %w", err)
	}
	return v, nil
}
