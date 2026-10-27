package bitmart

import (
	"context"
	"fmt"

	core_config "github.com/MIZHT/inter-exchange-arbitrage/internal/core/config"
	"github.com/MIZHT/inter-exchange-arbitrage/internal/core/domain"
	core_logger "github.com/MIZHT/inter-exchange-arbitrage/internal/core/logger"
	"github.com/MIZHT/inter-exchange-arbitrage/internal/core/transport/http_client"
	"github.com/MIZHT/inter-exchange-arbitrage/internal/core/transport/websocket_manager"
	"github.com/MIZHT/inter-exchange-arbitrage/internal/exchange"
	rest "github.com/MIZHT/inter-exchange-arbitrage/internal/exchange/bitmart/rest"
	bitmart_ws "github.com/MIZHT/inter-exchange-arbitrage/internal/exchange/bitmart/websocket"
)

type Bitmart struct {
	log  *core_logger.Logger
	rest *rest.Client
	ws   *bitmart_ws.Manager
}

func NewBitmart(log *core_logger.Logger, exCfg core_config.Exchange, proxy string, pools ...*websocket_manager.ProxyPool) *Bitmart {
	hc := http_client.NewHTTPClient(http_client.WithLogger(log.Logger), http_client.WithProxy(proxy))
	return &Bitmart{log: log, ws: bitmart_ws.NewManager(proxy, pools...), rest: rest.NewClient(hc, log, exCfg)}
}

func (ex *Bitmart) StartOrderBookStream(ctx context.Context, handlers exchange.OrderBookHandlers) error {
	err := ex.ws.Start(ctx, func(market websocket_manager.Market, book domain.OrderBook) {
		if market == websocket_manager.Spot && handlers.Spot != nil {
			handlers.Spot(book)
		} else if market == websocket_manager.Futures && handlers.Futures != nil {
			handlers.Futures(book)
		}
	})
	if err != nil {
		return fmt.Errorf("start BitMart order book streams: %w", err)
	}
	return nil
}
func (ex *Bitmart) SubscribeSpotOrderBooks(c context.Context, coins []string) error {
	return ex.ws.SubscribeSpot(c, coins)
}
func (ex *Bitmart) UnsubscribeSpotOrderBooks(c context.Context, coins []string) error {
	return ex.ws.UnsubscribeSpot(c, coins)
}
func (ex *Bitmart) SubscribeFuturesOrderBooks(c context.Context, coins []string) error {
	return ex.ws.SubscribeFutures(c, coins)
}
func (ex *Bitmart) UnsubscribeFuturesOrderBooks(c context.Context, coins []string) error {
	return ex.ws.UnsubscribeFutures(c, coins)
}
func (ex *Bitmart) ReconcileOrderBooks(c context.Context, spot, futures []string) error {
	return ex.ws.ReconcileOrderBooks(c, spot, futures)
}
func (ex *Bitmart) FetchCurrencies(ctx context.Context) ([]domain.Currency, error) {
	v, err := ex.rest.FetchCurrencies(ctx)
	if err != nil {
		return nil, fmt.Errorf("bitmart fetch currencies: %w", err)
	}
	return v, nil
}
func (ex *Bitmart) FetchMarkets(ctx context.Context) ([]domain.Market, error) {
	v, err := ex.rest.FetchMarkets(ctx)
	if err != nil {
		return nil, fmt.Errorf("bitmart fetch markets: %w", err)
	}
	return v, nil
}
