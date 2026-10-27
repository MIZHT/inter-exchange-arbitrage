package binance

import (
	"context"
	"fmt"

	core_config "github.com/MIZHT/inter-exchange-arbitrage/internal/core/config"
	"github.com/MIZHT/inter-exchange-arbitrage/internal/core/domain"
	core_logger "github.com/MIZHT/inter-exchange-arbitrage/internal/core/logger"
	"github.com/MIZHT/inter-exchange-arbitrage/internal/core/transport/http_client"
	"github.com/MIZHT/inter-exchange-arbitrage/internal/core/transport/websocket_manager"
	"github.com/MIZHT/inter-exchange-arbitrage/internal/exchange"
	binance_rest "github.com/MIZHT/inter-exchange-arbitrage/internal/exchange/binance/rest"
	binance_ws "github.com/MIZHT/inter-exchange-arbitrage/internal/exchange/binance/websocket"
)

type Binance struct {
	log  *core_logger.Logger
	rest *binance_rest.Client
	ws   *binance_ws.Manager
}

func NewBinance(
	log *core_logger.Logger,
	exCfg core_config.Exchange,
	proxy string,
	pools ...*websocket_manager.ProxyPool,
) *Binance {
	httpClient := http_client.NewHTTPClient(
		http_client.WithLogger(log.Logger),
		http_client.WithProxy(proxy),
	)

	return &Binance{
		log:  log,
		rest: binance_rest.NewClient(httpClient, log, exCfg),
		ws:   binance_ws.NewManager(proxy, pools...),
	}
}

func (ex *Binance) StartOrderBookStream(ctx context.Context, handlers exchange.OrderBookHandlers) error {
	err := ex.ws.Start(ctx, func(market websocket_manager.Market, book domain.OrderBook) {
		if market == websocket_manager.Spot && handlers.Spot != nil {
			handlers.Spot(book)
		} else if market == websocket_manager.Futures && handlers.Futures != nil {
			handlers.Futures(book)
		}
	})
	if err != nil {
		return fmt.Errorf("start Binance order book streams: %w", err)
	}
	return nil
}

func (ex *Binance) SubscribeSpotOrderBooks(ctx context.Context, coins []string) error {
	return ex.ws.SubscribeSpot(ctx, coins)
}
func (ex *Binance) UnsubscribeSpotOrderBooks(ctx context.Context, coins []string) error {
	return ex.ws.UnsubscribeSpot(ctx, coins)
}
func (ex *Binance) SubscribeFuturesOrderBooks(ctx context.Context, coins []string) error {
	return ex.ws.SubscribeFutures(ctx, coins)
}
func (ex *Binance) UnsubscribeFuturesOrderBooks(ctx context.Context, coins []string) error {
	return ex.ws.UnsubscribeFutures(ctx, coins)
}
func (ex *Binance) ReconcileOrderBooks(ctx context.Context, spot, futures []string) error {
	return ex.ws.ReconcileOrderBooks(ctx, spot, futures)
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
