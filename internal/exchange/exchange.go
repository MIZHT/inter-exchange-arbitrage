package exchange

import (
	"context"

	"github.com/MIZHT/inter-exchange-arbitrage/internal/core/domain"
)

type ExchangeAdapter interface {
	Name() string

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

	SubscribeOrderBook(context.Context, string) error
	UnsubscribeOrderBook(context.Context, string) error

	//Start(ctx context.Context) error
	//Stop(ctx context.Context) error
}
