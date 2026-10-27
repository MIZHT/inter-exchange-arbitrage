package exchange

import (
	"context"

	"github.com/MIZHT/inter-exchange-arbitrage/internal/core/domain"
)

type ExchangeAdapter interface {
	FetchCurrencies(context.Context) ([]domain.Currency, error)
	// need:
	// 0. Name without "/USDT" and CapsLock
	// 1. Networks > 0
	// 2. NetworkInfo.DepositEnabled || NetworkInfo.WithdrawEnabled
	// 3. WithdrawFee (кроме gateio) float64
	// 4. addr != "" && ToLower

	FetchMarkets(context.Context) ([]domain.Market, error)
	// need:
	// Name with "/USDT" and CapsLock
	// Spotable || Futurable
	// if FuturesNextFundingTime > 0 && (FuturesFundingRate == "")
	//

	StartOrderBookStream(context.Context, OrderBookHandlers) error

	SubscribeSpotOrderBooks(context.Context, []string) error
	UnsubscribeSpotOrderBooks(context.Context, []string) error

	SubscribeFuturesOrderBooks(context.Context, []string) error
	UnsubscribeFuturesOrderBooks(context.Context, []string) error
}

// OrderBookSubscriptionReconciler replaces the current subscriptions with the
// matcher result in one operation.
type OrderBookSubscriptionReconciler interface {
	ReconcileOrderBooks(context.Context, []string, []string) error
}

type OrderBookHandlers struct {
	Spot    func(domain.OrderBook)
	Futures func(domain.OrderBook)
}
