package bitget

import (
	"context"
	"fmt"

	core_config "github.com/MIZHT/inter-exchange-arbitrage/internal/core/config"
	"github.com/MIZHT/inter-exchange-arbitrage/internal/core/domain"
	core_logger "github.com/MIZHT/inter-exchange-arbitrage/internal/core/logger"
	"github.com/MIZHT/inter-exchange-arbitrage/internal/core/transport/http_client"
	"github.com/MIZHT/inter-exchange-arbitrage/internal/core/transport/websocket_manager"
	"github.com/MIZHT/inter-exchange-arbitrage/internal/exchange"
	rest "github.com/MIZHT/inter-exchange-arbitrage/internal/exchange/bitget/rest"
	bitget_ws "github.com/MIZHT/inter-exchange-arbitrage/internal/exchange/bitget/websocket"
)

type Bitget struct {
	log  *core_logger.Logger
	rest *rest.Client
	ws   *bitget_ws.Manager
}

func NewBitget(log *core_logger.Logger,
	exCfg core_config.Exchange,
	proxy string,
	pools ...*websocket_manager.ProxyPool,
) *Bitget {
	hc := http_client.NewHTTPClient(
		http_client.WithLogger(log.Logger),
		http_client.WithProxy(proxy),
	)
	return &Bitget{log: log,
		rest: rest.NewClient(hc, log, exCfg),
		ws:   bitget_ws.NewManager(proxy, pools...)}
}

func (ex *Bitget) StartOrderBookStream(ctx context.Context, handlers exchange.OrderBookHandlers) error {
	err := ex.ws.Start(ctx, func(market websocket_manager.Market, book domain.OrderBook) {
		if market == websocket_manager.Spot && handlers.Spot != nil {
			handlers.Spot(book)
		} else if market == websocket_manager.Futures && handlers.Futures != nil {
			handlers.Futures(book)
		}
	})
	if err != nil {
		return fmt.Errorf("start Bitget order book streams: %w", err)
	}
	return nil
}
func (ex *Bitget) SubscribeSpotOrderBooks(c context.Context, coins []string) error {
	return ex.ws.SubscribeSpot(c, coins)
}
func (ex *Bitget) UnsubscribeSpotOrderBooks(c context.Context, coins []string) error {
	return ex.ws.UnsubscribeSpot(c, coins)
}
func (ex *Bitget) SubscribeFuturesOrderBooks(c context.Context, coins []string) error {
	return ex.ws.SubscribeFutures(c, coins)
}
func (ex *Bitget) UnsubscribeFuturesOrderBooks(c context.Context, coins []string) error {
	return ex.ws.UnsubscribeFutures(c, coins)
}
func (ex *Bitget) ReconcileOrderBooks(c context.Context, spot, futures []string) error {
	return ex.ws.ReconcileOrderBooks(c, spot, futures)
}
func (ex *Bitget) FetchCurrencies(ctx context.Context) ([]domain.Currency, error) {
	v, err := ex.rest.FetchCurrencies(ctx)
	if err != nil {
		return nil, fmt.Errorf("bitget fetch currencies: %w", err)
	}
	return v, nil
}
func (ex *Bitget) FetchMarkets(ctx context.Context) ([]domain.Market, error) {
	v, err := ex.rest.FetchMarkets(ctx)
	if err != nil {
		return nil, fmt.Errorf("bitget fetch markets: %w", err)
	}
	return v, nil
}
