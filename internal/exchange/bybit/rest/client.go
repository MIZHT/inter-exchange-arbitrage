package rest_bybit

import (
	"context"

	"github.com/MIZHT/inter-exchange-arbitrage/internal/core/domain"
	"github.com/MIZHT/inter-exchange-arbitrage/internal/core/transport/http_client"
)

func FetchCurrencies(ctx context.Context, client *http_client.Client) ([]domain.Currency, error) {
	return []domain.Currency{}, nil
}

func FetchMarkets(ctx context.Context, client *http_client.Client) ([]domain.Market, error) {
	return []domain.Market{}, nil
}
