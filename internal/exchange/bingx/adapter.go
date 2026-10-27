package bingx

import (
	"context"
	"fmt"

	core_config "github.com/MIZHT/inter-exchange-arbitrage/internal/core/config"
	"github.com/MIZHT/inter-exchange-arbitrage/internal/core/domain"
	core_logger "github.com/MIZHT/inter-exchange-arbitrage/internal/core/logger"
	"github.com/MIZHT/inter-exchange-arbitrage/internal/core/transport/http_client"
	"github.com/MIZHT/inter-exchange-arbitrage/internal/core/transport/websocket_manager"
	"github.com/MIZHT/inter-exchange-arbitrage/internal/exchange"
	bingx_rest "github.com/MIZHT/inter-exchange-arbitrage/internal/exchange/bingx/rest"
	bingx_ws "github.com/MIZHT/inter-exchange-arbitrage/internal/exchange/bingx/websocket"
)

type Bingx struct {
	log  *core_logger.Logger
	rest *bingx_rest.Client
	ws   *bingx_ws.Manager
}

func NewBingx(
	log *core_logger.Logger,
	exCfg core_config.Exchange,
	proxy string,
	pools ...*websocket_manager.ProxyPool,
) *Bingx {
	httpClient := http_client.NewHTTPClient(
		http_client.WithLogger(log.Logger),
		http_client.WithProxy(proxy),
	)

	return &Bingx{
		log:  log,
		rest: bingx_rest.NewClient(httpClient, log, exCfg),
		ws:   bingx_ws.NewManager(proxy, pools...),
	}
}

func (ex *Bingx) StartOrderBookStream(ctx context.Context, handlers exchange.OrderBookHandlers) error {
	err := ex.ws.Start(ctx, func(market websocket_manager.Market, book domain.OrderBook) {
		if market == websocket_manager.Spot && handlers.Spot != nil {
			handlers.Spot(book)
		} else if market == websocket_manager.Futures && handlers.Futures != nil {
			handlers.Futures(book)
		}
	})
	if err != nil {
		return fmt.Errorf("start BingX order book streams: %w", err)
	}
	return nil
}
func (ex *Bingx) SubscribeSpotOrderBooks(c context.Context, coins []string) error {
	return ex.ws.SubscribeSpot(c, coins)
}
func (ex *Bingx) UnsubscribeSpotOrderBooks(c context.Context, coins []string) error {
	return ex.ws.UnsubscribeSpot(c, coins)
}
func (ex *Bingx) SubscribeFuturesOrderBooks(c context.Context, coins []string) error {
	return ex.ws.SubscribeFutures(c, coins)
}
func (ex *Bingx) UnsubscribeFuturesOrderBooks(c context.Context, coins []string) error {
	return ex.ws.UnsubscribeFutures(c, coins)
}
func (ex *Bingx) ReconcileOrderBooks(c context.Context, spot, futures []string) error {
	return ex.ws.ReconcileOrderBooks(c, spot, futures)
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
