package cmc

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	core_logger "github.com/MIZHT/inter-exchange-arbitrage/internal/core/logger"
	"github.com/MIZHT/inter-exchange-arbitrage/internal/core/transport/http_client"
)

// добавить прокси
// блокировка изменения snapshot через стакан

func RunCmc(
	ctx context.Context,
	log *core_logger.Logger,
	proxies []string,
	store *Store,
) error {
	if len(proxies) < 1 {
		return fmt.Errorf("no proxies for cmc")
	}

	symbolsList, err := FetchIndex(ctx, http_client.NewHTTPClient())
	if err != nil {
		log.Error("fetch CMC index failed",
			slog.Any("error", err),
		)
		return fmt.Errorf("fetch CMC index: %w", err)
	}

	var mu sync.Mutex
	snapshot := make(Snapshot)

	jobs := make(chan int, len(proxies))
	var wg sync.WaitGroup

	var success int
	var failed int

	for _, proxy := range proxies {
		wg.Add(1)

		go func(proxy string) {
			defer wg.Done()

			new_client := http_client.NewHTTPClient(
				http_client.WithProxy(proxy),
				http_client.WithLogger(log.Logger))

			for coinID := range jobs {
				if ctx.Err() != nil {
					return
				}

				listMarkets, symbol, err := FetchMarketPairs(ctx, new_client, coinID)
				if err != nil {
					log.Warn("fetch market pairs",
						slog.Int("coinID", coinID),
						slog.Any("err", err.Error()),
					)

					mu.Lock()
					failed++
					mu.Unlock()

					continue
				}

				mapTicker := MapTicker(coinID, listMarkets)

				mu.Lock()
				success++
				makeSnapshot(symbol, mapTicker, snapshot)
				mu.Unlock()

			}
		}(proxy)

	}
	go func() {
		defer close(jobs)
		for _, coinID := range symbolsList {
			select {
			case jobs <- coinID:
			case <-ctx.Done():
				return
			}
		}

	}()

	wg.Wait()

	store.publish(&snapshot)

	log.Info("cmc snapshot update",
		slog.Int("total coins", len(symbolsList)),
		slog.Int("success", success),
		slog.Int("failed", failed),
		slog.Int("snapshot_symbols", len(snapshot)))

	return nil

}

func NewStore() *Store {
	s := &Store{}
	empty := Snapshot{}
	s.cur.Store(&empty) // чтобы Get никогда не вернул nil
	return s
}

func (s *Store) Get() *Snapshot { return s.cur.Load() }

func (s *Store) publish(snap *Snapshot) { s.cur.Store(snap) }
