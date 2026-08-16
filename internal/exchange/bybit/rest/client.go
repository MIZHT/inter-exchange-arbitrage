package rest_bybit

import (
	"context"

	"github.com/MIZHT/inter-exchange-arbitrage/internal/core/domain"
)

func FetchCurrencies(ctx context.Context) ([]domain.Currency, error) {
	return []domain.Currency{}, nil
}

func FetchMarkets(ctx context.Context) ([]domain.Market, error) {
	return []domain.Market{}, nil
}
