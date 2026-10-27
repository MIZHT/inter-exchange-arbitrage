package bybit

import (
	"context"
	"fmt"

	core_config "github.com/MIZHT/inter-exchange-arbitrage/internal/core/config"
	"github.com/MIZHT/inter-exchange-arbitrage/internal/core/domain"
	core_logger "github.com/MIZHT/inter-exchange-arbitrage/internal/core/logger"
	"github.com/MIZHT/inter-exchange-arbitrage/internal/core/transport/http_client"
	"github.com/MIZHT/inter-exchange-arbitrage/internal/core/transport/websocket_manager"
	"github.com/MIZHT/inter-exchange-arbitrage/internal/exchange"
	rest "github.com/MIZHT/inter-exchange-arbitrage/internal/exchange/bybit/rest"
	bybit_ws "github.com/MIZHT/inter-exchange-arbitrage/internal/exchange/bybit/websocket"
)

type Bybit struct {
	log  *core_logger.Logger
	rest *rest.Client
	ws   *bybit_ws.Manager
}

func NewBybit(log *core_logger.Logger, exCfg core_config.Exchange, proxy string, pools ...*websocket_manager.ProxyPool) *Bybit {
	hc := http_client.NewHTTPClient(http_client.WithLogger(log.Logger), http_client.WithProxy(proxy))
	return &Bybit{log: log, ws: bybit_ws.NewManager(proxy, pools...), rest: rest.NewClient(hc, log, exCfg)}
}

func (ex *Bybit) StartOrderBookStream(ctx context.Context, handlers exchange.OrderBookHandlers) error {
	err := ex.ws.Start(ctx, func(market websocket_manager.Market, book domain.OrderBook) {
		if market == websocket_manager.Spot && handlers.Spot != nil {
			handlers.Spot(book)
		} else if market == websocket_manager.Futures && handlers.Futures != nil {
			handlers.Futures(book)
		}
	})
	if err != nil {
		return fmt.Errorf("start Bybit order book streams: %w", err)
	}
	return nil
}
func (ex *Bybit) SubscribeSpotOrderBooks(c context.Context, coins []string) error {
	return ex.ws.SubscribeSpot(c, coins)
}
func (ex *Bybit) UnsubscribeSpotOrderBooks(c context.Context, coins []string) error {
	return ex.ws.UnsubscribeSpot(c, coins)
}
func (ex *Bybit) SubscribeFuturesOrderBooks(c context.Context, coins []string) error {
	return ex.ws.SubscribeFutures(c, coins)
}
func (ex *Bybit) UnsubscribeFuturesOrderBooks(c context.Context, coins []string) error {
	return ex.ws.UnsubscribeFutures(c, coins)
}
func (ex *Bybit) ReconcileOrderBooks(c context.Context, spot, futures []string) error {
	return ex.ws.ReconcileOrderBooks(c, spot, futures)
}
func (ex *Bybit) FetchCurrencies(ctx context.Context) ([]domain.Currency, error) {
	v, err := ex.rest.FetchCurrencies(ctx)
	if err != nil {
		return nil, fmt.Errorf("bybit fetch currencies: %w", err)
	}
	return v, nil
}
func (ex *Bybit) FetchMarkets(ctx context.Context) ([]domain.Market, error) {
	v, err := ex.rest.FetchMarkets(ctx)
	if err != nil {
		return nil, fmt.Errorf("bybit fetch markets: %w", err)
	}
	return v, nil
}
