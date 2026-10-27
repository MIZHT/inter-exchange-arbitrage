package engine

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"

	core_config "github.com/MIZHT/inter-exchange-arbitrage/internal/core/config"
	"github.com/MIZHT/inter-exchange-arbitrage/internal/core/domain"
	"github.com/MIZHT/inter-exchange-arbitrage/internal/exchange"
)

type ExchangeOrderBooks map[core_config.ExchangeName]map[string]domain.OrderBook

type OrderBookSnapshot struct {
	Spot    ExchangeOrderBooks
	Futures ExchangeOrderBooks
}

type exchangeOrderBooks struct {
	mu      sync.RWMutex
	spot    map[string]domain.OrderBook
	futures map[string]domain.OrderBook
}

type OrderBookStore struct {
	mu        sync.RWMutex
	exchanges map[core_config.ExchangeName]*exchangeOrderBooks
}

func NewOrderBookStore() *OrderBookStore {
	return &OrderBookStore{exchanges: make(map[core_config.ExchangeName]*exchangeOrderBooks)}
}

func (s *OrderBookStore) exchange(name core_config.ExchangeName) *exchangeOrderBooks {
	s.mu.RLock()
	books := s.exchanges[name]
	s.mu.RUnlock()
	if books != nil {
		return books
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if books = s.exchanges[name]; books == nil {
		books = &exchangeOrderBooks{spot: make(map[string]domain.OrderBook), futures: make(map[string]domain.OrderBook)}
		s.exchanges[name] = books
	}
	return books
}

type CoinSet map[string]struct{}

type OrderBookRunner struct {
	exchangeName      core_config.ExchangeName
	adapter           exchange.ExchangeAdapter
	store             *OrderBookStore
	mu                sync.Mutex
	started           bool
	spotSubscribed    CoinSet
	futuresSubscribed CoinSet
}

func NewOrderBookRunner(adapter exchange.ExchangeAdapter, name core_config.ExchangeName, store *OrderBookStore) *OrderBookRunner {
	return &OrderBookRunner{exchangeName: name, adapter: adapter, store: store, spotSubscribed: make(CoinSet), futuresSubscribed: make(CoinSet)}
}

func cloneBookMap(source map[string]domain.OrderBook) map[string]domain.OrderBook {
	result := make(map[string]domain.OrderBook, len(source))
	for coin, book := range source {
		book.Bids = append([]domain.BidAsk(nil), book.Bids...)
		book.Asks = append([]domain.BidAsk(nil), book.Asks...)
		domain.NormalizeOrderBook(&book)
		result[coin] = book
	}
	return result
}

func (s *OrderBookStore) Snapshot() OrderBookSnapshot {
	s.mu.RLock()
	exchanges := make(map[core_config.ExchangeName]*exchangeOrderBooks, len(s.exchanges))
	for name, books := range s.exchanges {
		exchanges[name] = books
	}
	s.mu.RUnlock()

	snapshot := OrderBookSnapshot{Spot: make(ExchangeOrderBooks), Futures: make(ExchangeOrderBooks)}
	for name, books := range exchanges {
		books.mu.RLock()
		spot := cloneBookMap(books.spot)
		futures := cloneBookMap(books.futures)
		books.mu.RUnlock()
		if len(spot) > 0 {
			snapshot.Spot[name] = spot
		}
		if len(futures) > 0 {
			snapshot.Futures[name] = futures
		}
	}
	return snapshot
}

func (r *OrderBookRunner) Start(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.started {
		return nil
	}
	err := r.adapter.StartOrderBookStream(ctx, exchange.OrderBookHandlers{
		Spot: func(book domain.OrderBook) {
			r.store.PutSpot(r.exchangeName, book)
		},
		Futures: func(book domain.OrderBook) {
			r.store.PutFutures(r.exchangeName, book)
		},
	})
	if err != nil {
		return fmt.Errorf("start %s order book stream: %w", r.exchangeName, err)
	}
	r.started = true
	return nil
}

func prepareBook(book domain.OrderBook) domain.OrderBook {
	book.Bids = append([]domain.BidAsk(nil), book.Bids...)
	book.Asks = append([]domain.BidAsk(nil), book.Asks...)
	domain.NormalizeOrderBook(&book)
	return book
}

func (s *OrderBookStore) PutSpot(exchangeName core_config.ExchangeName, book domain.OrderBook) {
	books := s.exchange(exchangeName)
	book = prepareBook(book)
	books.mu.Lock()
	books.spot[book.Market] = book
	books.mu.Unlock()
}
func (s *OrderBookStore) PutFutures(exchangeName core_config.ExchangeName, book domain.OrderBook) {
	books := s.exchange(exchangeName)
	book = prepareBook(book)
	books.mu.Lock()
	books.futures[book.Market] = book
	books.mu.Unlock()
}

func (r *OrderBookRunner) UpdateCoins(ctx context.Context, spotCoins, futureCoins CoinSet) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.started {
		return nil
	}
	if reconciler, ok := r.adapter.(exchange.OrderBookSubscriptionReconciler); ok {
		removedSpot := difference(r.spotSubscribed, spotCoins)
		removedFutures := difference(r.futuresSubscribed, futureCoins)
		if err := reconciler.ReconcileOrderBooks(ctx, coinSetSlice(spotCoins), coinSetSlice(futureCoins)); err != nil {
			return err
		}
		r.store.DeleteSpot(r.exchangeName, removedSpot)
		r.store.DeleteFutures(r.exchangeName, removedFutures)
		r.spotSubscribed = cloneCoinSet(spotCoins)
		r.futuresSubscribed = cloneCoinSet(futureCoins)
		return nil
	}
	var errs []error
	if err := r.updateCoins(ctx,
		spotCoins,
		r.spotSubscribed,
		r.adapter.SubscribeSpotOrderBooks,
		r.adapter.UnsubscribeSpotOrderBooks,
		r.store.DeleteSpot,
	); err != nil {

		errs = append(errs, err)
	}
	if err := r.updateCoins(ctx,
		futureCoins,
		r.futuresSubscribed,
		r.adapter.SubscribeFuturesOrderBooks,
		r.adapter.UnsubscribeFuturesOrderBooks,
		r.store.DeleteFutures,
	); err != nil {

		errs = append(errs, err)
	}
	return errors.Join(errs...)
}

func coinSetSlice(coins CoinSet) []string {
	result := make([]string, 0, len(coins))
	for coin := range coins {
		result = append(result, coin)
	}
	sort.Strings(result)
	return result
}

func cloneCoinSet(coins CoinSet) CoinSet {
	result := make(CoinSet, len(coins))
	for coin := range coins {
		result[coin] = struct{}{}
	}
	return result
}

func difference(left, right CoinSet) []string {
	var result []string
	for coin := range left {
		if _, exists := right[coin]; !exists {
			result = append(result, coin)
		}
	}
	return result
}

type subscribeFunc func(context.Context, []string) error
type deleteFunc func(core_config.ExchangeName, []string)

func (r *OrderBookRunner) updateCoins(ctx context.Context,
	desired, current CoinSet,
	subscribe, unsubscribe subscribeFunc,
	deleteBooks deleteFunc,
) error {
	var add, remove []string
	for coin := range desired {
		if _, ok := current[coin]; !ok {
			add = append(add, coin)
		}
	}
	for coin := range current {
		if _, ok := desired[coin]; !ok {
			remove = append(remove, coin)
		}
	}
	var errs []error
	if len(add) > 0 {
		if err := subscribe(ctx, add); err != nil {
			errs = append(errs, fmt.Errorf("subscribe order books on %s: %w", r.exchangeName, err))
		} else {
			for _, coin := range add {
				current[coin] = struct{}{}
			}
		}
	}
	if len(remove) > 0 && len(errs) == 0 {
		if err := unsubscribe(ctx, remove); err != nil {
			errs = append(errs, fmt.Errorf("unsubscribe order books on %s: %w", r.exchangeName, err))
		} else {
			for _, coin := range remove {
				delete(current, coin)
			}
			deleteBooks(r.exchangeName, remove)
		}
	}
	return errors.Join(errs...)
}

func deleteBooks(books map[string]domain.OrderBook, coins []string) {
	for _, coin := range coins {
		delete(books, coin)
	}
}
func (s *OrderBookStore) DeleteSpot(exchangeName core_config.ExchangeName, coins []string) {
	books := s.exchange(exchangeName)
	books.mu.Lock()
	deleteBooks(books.spot, coins)
	books.mu.Unlock()
}
func (s *OrderBookStore) DeleteFutures(exchangeName core_config.ExchangeName, coins []string) {
	books := s.exchange(exchangeName)
	books.mu.Lock()
	deleteBooks(books.futures, coins)
	books.mu.Unlock()
}
