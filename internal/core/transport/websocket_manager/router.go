package websocket_manager

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/MIZHT/inter-exchange-arbitrage/internal/core/domain"
	"github.com/gorilla/websocket"
)

type Market string

const (
	Spot    Market = "spot"
	Futures Market = "futures"
)

type Update struct {
	Book     domain.OrderBook
	Snapshot bool
}
type Protocol struct {
	Name         string
	Endpoint     func(Market) string
	Capacity     func(Market) int
	Subscribe    func(Market, []string, bool) []any
	Decode       func([]byte) []byte
	Parse        func(Market, []byte) (Update, bool)
	Compression  func(Market) bool
	PingInterval time.Duration
	PingMessage  func(Market) any
	PingText     bool
	Pong         func(Market, []byte) ([]byte, bool)
}

type groupConnection struct {
	client  atomic.Pointer[Client]
	stateMu sync.RWMutex
	closed  bool
	proxy   string
	market  Market
	id      string
	coins   map[string]struct{}
}

func (c *groupConnection) readMessage() ([]byte, error) {
	client := c.client.Load()
	if client == nil {
		return nil, fmt.Errorf("WebSocket connection is not open")
	}
	return client.ReadMessage()
}

func (c *groupConnection) writeJSON(value any) error {
	c.stateMu.RLock()
	defer c.stateMu.RUnlock()
	if c.closed {
		return fmt.Errorf("WebSocket connection is closed")
	}
	client := c.client.Load()
	if client == nil {
		return fmt.Errorf("WebSocket connection is not open")
	}
	return client.WriteJSON(value)
}

func (c *groupConnection) writeMessage(messageType int, payload []byte) error {
	c.stateMu.RLock()
	defer c.stateMu.RUnlock()
	if c.closed {
		return fmt.Errorf("WebSocket connection is closed")
	}
	client := c.client.Load()
	if client == nil {
		return fmt.Errorf("WebSocket connection is not open")
	}
	return client.WriteMessage(messageType, payload)
}

func (c *groupConnection) coinList() []string {
	c.stateMu.RLock()
	defer c.stateMu.RUnlock()
	coins := make([]string, 0, len(c.coins))
	for coin := range c.coins {
		coins = append(coins, coin)
	}
	sort.Strings(coins)
	return coins
}

func (c *groupConnection) setCoins(coins map[string]struct{}) {
	c.stateMu.Lock()
	c.coins = make(map[string]struct{}, len(coins))
	for coin := range coins {
		c.coins[coin] = struct{}{}
	}
	c.stateMu.Unlock()
}

func (c *groupConnection) isClosed() bool {
	c.stateMu.RLock()
	defer c.stateMu.RUnlock()
	return c.closed
}

func (c *groupConnection) close() {
	c.stateMu.Lock()
	c.closed = true
	client := c.client.Load()
	c.stateMu.Unlock()
	if client != nil {
		_ = client.Close()
	}
}

type Router struct {
	protocol    Protocol
	planner     map[Market]*GroupPlanner
	mu          sync.Mutex
	connections map[string]*groupConnection
	stateMu     sync.Mutex
	books       map[Market]map[string]domain.OrderBook
	started     bool
	onBook      func(Market, domain.OrderBook)
}

func NewRouter(protocol Protocol, pool *ProxyPool) *Router {
	if pool == nil {
		pool = NewProxyPool(nil)
	}
	r := &Router{protocol: protocol, connections: make(map[string]*groupConnection), books: map[Market]map[string]domain.OrderBook{Spot: {}, Futures: {}}, planner: make(map[Market]*GroupPlanner)}
	for _, market := range []Market{Spot, Futures} {
		capacity := protocol.Capacity(market)
		r.planner[market] = NewGroupPlanner(pool, protocol.Name, string(market), capacity)
	}
	return r
}

func (r *Router) Start(_ context.Context, onBook func(Market, domain.OrderBook)) error {
	r.mu.Lock()
	r.onBook = onBook
	r.started = true
	r.mu.Unlock()
	return nil
}

func (r *Router) ReconcileOrderBooks(ctx context.Context, spotCoins, futuresCoins []string) error {
	for _, item := range []struct {
		market Market
		coins  []string
	}{{Spot, spotCoins}, {Futures, futuresCoins}} {
		r.planner[item.market].Set(item.coins)
		if err := r.reconcile(ctx, item.market); err != nil {
			return err
		}
	}
	return nil
}

func (r *Router) SubscribeSpot(ctx context.Context, coins []string) error {
	return r.subscribe(ctx, Spot, coins, true)
}
func (r *Router) UnsubscribeSpot(ctx context.Context, coins []string) error {
	return r.subscribe(ctx, Spot, coins, false)
}
func (r *Router) SubscribeFutures(ctx context.Context, coins []string) error {
	return r.subscribe(ctx, Futures, coins, true)
}
func (r *Router) UnsubscribeFutures(ctx context.Context, coins []string) error {
	return r.subscribe(ctx, Futures, coins, false)
}

func (r *Router) subscribe(ctx context.Context, market Market, coins []string, add bool) error {
	r.mu.Lock()
	started := r.started
	r.mu.Unlock()
	if !started {
		return fmt.Errorf("%s %s WebSocket router not started", r.protocol.Name, market)
	}
	if add {
		r.planner[market].Add(coins)
		return r.reconcile(ctx, market)
	}
	r.planner[market].Remove(coins)
	r.stateMu.Lock()
	for _, coin := range coins {
		delete(r.books[market], coin)
	}
	r.stateMu.Unlock()
	return r.reconcile(ctx, market)
}

func (r *Router) reconcile(ctx context.Context, market Market) error {
	for _, group := range r.planner[market].All() {
		r.mu.Lock()
		conn := r.connections[group.ID]
		r.mu.Unlock()
		if len(group.Coins) == 0 {
			if conn != nil {
				r.mu.Lock()
				delete(r.connections, group.ID)
				r.mu.Unlock()
				conn.close()
			}
			continue
		}
		if conn == nil {
			if err := r.ensureGroup(ctx, market, group.ID, group.Proxy); err != nil {
				return err
			}
			r.mu.Lock()
			conn = r.connections[group.ID]
			r.mu.Unlock()
		}
		var add, remove []string
		currentCoins := conn.coinList()
		current := make(map[string]struct{}, len(currentCoins))
		for _, coin := range currentCoins {
			current[coin] = struct{}{}
		}
		for coin := range group.Coins {
			if _, ok := current[coin]; !ok {
				add = append(add, coin)
			}
		}
		for coin := range current {
			if _, ok := group.Coins[coin]; !ok {
				remove = append(remove, coin)
			}
		}
		sort.Strings(add)
		sort.Strings(remove)
		if err := r.writeSubscription(conn, market, remove, false); err != nil {
			return err
		}
		if err := r.writeSubscription(conn, market, add, true); err != nil {
			return err
		}
		conn.setCoins(group.Coins)
	}
	return nil
}

func (r *Router) ensureGroup(ctx context.Context, market Market, id, proxy string) error {
	r.mu.Lock()
	if r.connections[id] != nil {
		r.mu.Unlock()
		return nil
	}
	r.mu.Unlock()
	client, err := Dial(ctx, r.protocol.Endpoint(market), proxy, r.protocol.compressionEnabled(market))
	if err != nil {
		return fmt.Errorf("dial %s %s group %s: %w", r.protocol.Name, market, id, err)
	}
	conn := &groupConnection{market: market, id: id, proxy: proxy, coins: make(map[string]struct{})}
	conn.client.Store(client)
	r.mu.Lock()
	if current := r.connections[id]; current != nil {
		r.mu.Unlock()
		_ = client.Close()
		return nil
	}
	r.connections[id] = conn
	r.mu.Unlock()
	go r.readLoop(ctx, conn)
	if r.protocol.PingInterval > 0 && r.protocol.PingMessage != nil {
		go r.pingLoop(ctx, conn)
	}
	return nil
}

func (r *Router) writeSubscription(conn *groupConnection, market Market, coins []string, add bool) error {
	if conn == nil || len(coins) == 0 {
		return nil
	}
	messages := r.protocol.Subscribe(market, coins, add)
	for _, message := range messages {
		if err := conn.writeJSON(message); err != nil {
			return fmt.Errorf("send %s subscription: %w", r.protocol.Name, err)
		}
	}
	return nil
}

func (r *Router) readLoop(ctx context.Context, conn *groupConnection) {
	defer conn.close()
	for {
		raw, err := conn.readMessage()
		if err != nil {
			if ctx.Err() != nil || conn.isClosed() || !r.reconnect(ctx, conn) {
				return
			}
			continue
		}
		if r.protocol.Decode != nil {
			raw = r.protocol.Decode(raw)
		}
		if r.protocol.Pong != nil {
			if response, handled := r.protocol.Pong(conn.market, raw); handled {
				if len(response) > 0 {
					_ = conn.writeMessage(websocket.TextMessage, response)
				}
				continue
			}
		}
		update, ok := r.protocol.Parse(conn.market, raw)
		if !ok {
			continue
		}
		book := update.Book
		if book.Market == "" {
			continue
		}
		book.Updated = time.Now()
		r.stateMu.Lock()
		state := r.books[conn.market]
		previous, exists := state[book.Market]
		if update.Snapshot {
			state[book.Market] = book
		} else if exists {
			previous.Bids = mergeLevels(previous.Bids, book.Bids, true)
			previous.Asks = mergeLevels(previous.Asks, book.Asks, false)
			previous.Updated = book.Updated
			state[book.Market] = previous
		} else {
			r.stateMu.Unlock()
			continue
		}
		current := state[book.Market]
		r.stateMu.Unlock()
		r.mu.Lock()
		onBook := r.onBook
		r.mu.Unlock()
		if onBook != nil {
			onBook(conn.market, current)
		}
	}
}

func (r *Router) reconnect(ctx context.Context, conn *groupConnection) bool {
	delay := time.Second
	for {
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return false
		case <-timer.C:
		}
		if conn.isClosed() {
			return false
		}
		client, err := Dial(ctx, r.protocol.Endpoint(conn.market), conn.proxy, r.protocol.compressionEnabled(conn.market))
		if err != nil {
			delay = nextRetryDelay(delay)
			continue
		}

		conn.stateMu.Lock()
		if conn.closed {
			conn.stateMu.Unlock()
			_ = client.Close()
			return false
		}
		coins := make([]string, 0, len(conn.coins))
		for coin := range conn.coins {
			coins = append(coins, coin)
		}
		sort.Strings(coins)
		var writeErr error
		if len(coins) > 0 {
			for _, message := range r.protocol.Subscribe(conn.market, coins, true) {
				if err := client.WriteJSON(message); err != nil {
					writeErr = err
					break
				}
			}
		}
		if writeErr != nil {
			conn.stateMu.Unlock()
			_ = client.Close()
			delay = nextRetryDelay(delay)
			continue
		}
		previous := conn.client.Swap(client)
		conn.stateMu.Unlock()
		if previous != nil {
			_ = previous.Close()
		}
		return true
	}
}

func (p Protocol) compressionEnabled(market Market) bool {
	return p.Compression != nil && p.Compression(market)
}

func nextRetryDelay(delay time.Duration) time.Duration {
	delay *= 2
	if delay > 30*time.Second {
		return 30 * time.Second
	}
	return delay
}

func (r *Router) pingLoop(ctx context.Context, conn *groupConnection) {
	ticker := time.NewTicker(r.protocol.PingInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			var err error
			if r.protocol.PingText {
				err = conn.writeMessage(websocket.TextMessage, []byte(r.protocol.PingMessage(conn.market).(string)))
			} else {
				err = conn.writeJSON(r.protocol.PingMessage(conn.market))
			}
			if err != nil {
				if ctx.Err() != nil || conn.isClosed() {
					return
				}
			}
		}
	}
}

func mergeLevels(current, updates []domain.BidAsk, bids bool) []domain.BidAsk {
	levels := make(map[float64]float64, len(current)+len(updates))
	for _, level := range current {
		levels[level.Price] = level.Quantity
	}
	for _, level := range updates {
		if level.Quantity <= 0 {
			delete(levels, level.Price)
		} else {
			levels[level.Price] = level.Quantity
		}
	}
	result := make([]domain.BidAsk, 0, len(levels))
	for price, quantity := range levels {
		result = append(result, domain.BidAsk{Price: price, Quantity: quantity})
	}
	sort.Slice(result, func(i, j int) bool {
		if bids {
			return result[i].Price > result[j].Price
		}
		return result[i].Price < result[j].Price
	})
	return result
}
