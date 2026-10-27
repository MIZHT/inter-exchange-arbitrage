package engine

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	core_config "github.com/MIZHT/inter-exchange-arbitrage/internal/core/config"
	"github.com/MIZHT/inter-exchange-arbitrage/internal/core/domain"
)

const marketSnapshotSchemaVersion = 1

type MarketSnapshot struct {
	SchemaVersion int                                                  `json:"schema_version"`
	SnapshotID    string                                               `json:"snapshot_id"`
	CapturedAt    time.Time                                            `json:"captured_at"`
	Matches       []domain.ContractMatch                               `json:"matches"`
	Spot          map[core_config.ExchangeName]map[string]SnapshotBook `json:"spot"`
	Futures       map[core_config.ExchangeName]map[string]SnapshotBook `json:"futures"`
}

type SnapshotBook struct {
	UpdatedAt time.Time       `json:"updated_at"`
	Bids      []SnapshotLevel `json:"bids"`
	Asks      []SnapshotLevel `json:"asks"`
}

type SnapshotLevel struct {
	Price    float64 `json:"price"`
	Quantity float64 `json:"quantity"`
}

type SnapshotPublisher struct {
	endpoint string
	client   *http.Client
}

func NewSnapshotPublisher(endpoint string) (*SnapshotPublisher, error) {
	endpoint = strings.TrimSpace(endpoint)
	if endpoint == "" {
		return nil, nil
	}

	parsed, err := url.ParseRequestURI(endpoint)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return nil, fmt.Errorf("invalid market snapshot URL %q", endpoint)
	}

	return &SnapshotPublisher{
		endpoint: endpoint,
		client:   &http.Client{Timeout: 4 * time.Second},
	}, nil
}

func (p *SnapshotPublisher) Publish(ctx context.Context, snapshot MarketSnapshot) (int, error) {
	body, err := json.Marshal(snapshot)
	if err != nil {
		return 0, fmt.Errorf("encode market snapshot: %w", err)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, p.endpoint, bytes.NewReader(body))
	if err != nil {
		return len(body), fmt.Errorf("create market snapshot request: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")

	response, err := p.client.Do(request)
	if err != nil {
		return len(body), fmt.Errorf("send market snapshot: %w", err)
	}
	defer response.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 64*1024))
	if response.StatusCode != http.StatusNoContent {
		return len(body), fmt.Errorf("snapshot receiver returned HTTP %d", response.StatusCode)
	}
	return len(body), nil
}

func buildMarketSnapshot(matches *domain.MatchResult, books OrderBookSnapshot, capturedAt time.Time) MarketSnapshot {
	capturedAt = capturedAt.UTC()
	snapshot := MarketSnapshot{
		SchemaVersion: marketSnapshotSchemaVersion,
		SnapshotID:    fmt.Sprintf("%d", capturedAt.UnixNano()),
		CapturedAt:    capturedAt,
		Matches:       make([]domain.ContractMatch, 0),
		Spot:          make(map[core_config.ExchangeName]map[string]SnapshotBook),
		Futures:       make(map[core_config.ExchangeName]map[string]SnapshotBook),
	}
	if matches == nil {
		return snapshot
	}

	coins := make([]string, 0, len(matches.Matches))
	for coin := range matches.Matches {
		coins = append(coins, coin)
	}
	sort.Strings(coins)
	for _, coin := range coins {
		match := matches.Matches[coin]
		if match == nil {
			continue
		}
		copyMatch := *match
		copyMatch.Exchanges = append([]domain.MatchedExchange(nil), match.Exchanges...)
		copyMatch.Pairs = append([]domain.ExchangePair(nil), match.Pairs...)
		snapshot.Matches = append(snapshot.Matches, copyMatch)
	}

	snapshot.Spot = snapshotBooks(matches.Spot, books.Spot)
	snapshot.Futures = snapshotBooks(matches.Futures, books.Futures)
	return snapshot
}

func snapshotBooks(
	matched map[core_config.ExchangeName]map[string]struct{},
	books ExchangeOrderBooks,
) map[core_config.ExchangeName]map[string]SnapshotBook {
	result := make(map[core_config.ExchangeName]map[string]SnapshotBook)
	for exchangeName, coins := range matched {
		for coin := range coins {
			book, ok := books[exchangeName][coin]
			if !ok {
				continue
			}
			if result[exchangeName] == nil {
				result[exchangeName] = make(map[string]SnapshotBook)
			}
			result[exchangeName][coin] = toSnapshotBook(book)
		}
	}
	return result
}

func toSnapshotBook(book domain.OrderBook) SnapshotBook {
	result := SnapshotBook{
		UpdatedAt: book.Updated,
		Bids:      make([]SnapshotLevel, len(book.Bids)),
		Asks:      make([]SnapshotLevel, len(book.Asks)),
	}
	for i, level := range book.Bids {
		result.Bids[i] = SnapshotLevel{Price: level.Price, Quantity: level.Quantity}
	}
	for i, level := range book.Asks {
		result.Asks[i] = SnapshotLevel{Price: level.Price, Quantity: level.Quantity}
	}
	return result
}
