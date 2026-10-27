package engine

import (
	"context"
	"log/slog"
	"sort"
	"sync"
	"time"

	"github.com/MIZHT/inter-exchange-arbitrage/internal/cmc"
	core_config "github.com/MIZHT/inter-exchange-arbitrage/internal/core/config"
	"github.com/MIZHT/inter-exchange-arbitrage/internal/core/domain"
	core_logger "github.com/MIZHT/inter-exchange-arbitrage/internal/core/logger"
	"github.com/MIZHT/inter-exchange-arbitrage/internal/exchange"
)

type MatchInput struct {
	Exchanges map[core_config.ExchangeName]*ExchangeSnapshot
	CMC       *cmc.Snapshot
}

type Coordinator struct {
	mu        sync.Mutex
	ctx       context.Context
	cancel    context.CancelFunc
	logger    *core_logger.Logger
	proxies   []string
	snapshots *SnapshotPublisher

	cmcStore     *cmc.Store
	matcherStore *domain.MatchStore
	orderBooks   *OrderBookStore

	runners          map[core_config.ExchangeName]*ExchangeRunner
	orderBookRunners map[core_config.ExchangeName]*OrderBookRunner
}

func NewCoordinator(
	ctx context.Context,
	logger *core_logger.Logger,
	adapters map[core_config.ExchangeName]exchange.ExchangeAdapter,
	snapshots *SnapshotPublisher,
) *Coordinator {
	cmcStore, err := cmc.NewStore(cmc.DefaultSnapshotPath)
	if err != nil {
		logger.Warn(
			"could not load persisted CMC snapshot; starting with an empty snapshot",
			slog.Any("error", err),
		)
	} else if snapshot := cmcStore.Get(); snapshot != nil && len(*snapshot) > 0 {
		logger.Info(
			"loaded persisted CMC snapshot",
			slog.Int("symbols", len(*snapshot)),
		)
	}

	orderBooks := NewOrderBookStore()
	c := &Coordinator{
		ctx:              ctx,
		logger:           logger,
		snapshots:        snapshots,
		runners:          make(map[core_config.ExchangeName]*ExchangeRunner),
		orderBooks:       orderBooks,
		orderBookRunners: make(map[core_config.ExchangeName]*OrderBookRunner),
		cmcStore:         cmcStore,
		matcherStore:     domain.NewMatches(),
	}
	for name, ex := range adapters {
		c.runners[name] = NewExchangeRunner(ex, name, logger)
		c.orderBookRunners[name] = NewOrderBookRunner(ex, name, orderBooks)
	}
	return c
}

func (c *Coordinator) Run(ctx context.Context) error {
	var wg sync.WaitGroup
	c.ctx, c.cancel = context.WithCancel(ctx)

	tickerExFetch := time.NewTicker(60 * time.Second)
	defer tickerExFetch.Stop()
	wg.Add(1)
	go func() {
		defer wg.Done()
		c.ExFetch()

	}()

	tickerCMC := time.NewTicker(30 * time.Minute)
	defer tickerCMC.Stop()
	var snapshotTicks <-chan time.Time
	if c.snapshots != nil {
		tickerSnapshots := time.NewTicker(5 * time.Second)
		defer tickerSnapshots.Stop()
		snapshotTicks = tickerSnapshots.C
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		c.CMCParse()

	}()

	wg.Wait()

	c.MatchFind()

	c.StartOrderBookStream()
	c.UpdateOrderBookStream()
	for {
		select {
		case <-c.ctx.Done():
			return nil
		case <-tickerExFetch.C:
			wg.Add(1)
			go func() {
				defer wg.Done()
				c.ExFetch()
				c.MatchFind()
				c.UpdateOrderBookStream()

			}()
		case <-tickerCMC.C:
			wg.Add(1)
			go func() {
				defer wg.Done()
				c.CMCParse()
				c.MatchFind()
				c.UpdateOrderBookStream()

			}()
		case <-snapshotTicks:
			c.publishOrderBookSnapshot()
		}
	}
}

func (c *Coordinator) ExFetch() {
	c.logger.Debug("start fetch exchanges")

	var wg sync.WaitGroup
	for exchangeName, runner := range c.runners {
		c.logger.Debug("start exchange fetch", slog.String("exchange", string(exchangeName)))
		wg.Add(1)
		go func() {
			defer wg.Done()
			runner.RunExFetch(c.ctx)
		}()
	}
	wg.Wait()
}

func (c *Coordinator) CMCParse() {
	c.logger.Debug("start cmc")

	if err := cmc.RunCmc(c.ctx, c.logger, c.proxies, c.cmcStore); err != nil {
		c.logger.Warn(
			"CMC refresh failed; keeping the last available snapshot",
			slog.Any("error", err),
		)
		return
	}
}

func (c *Coordinator) MatchFind() {
	input := MatchInput{
		Exchanges: make(map[core_config.ExchangeName]*ExchangeSnapshot, len(c.runners)),
		CMC:       c.cmcStore.Get(),
	}
	c.logger.Debug("start match")

	for name, runner := range c.runners {
		if snapshot := runner.snapshot.Load(); snapshot != nil {
			input.Exchanges[name] = snapshot
		}
	}

	c.RunMatchFind(input)
}

// func(c *Coordinator) FetchOrderbook() {
// 	SnapOrderbook:= make(map[core_config.ExchangeName]map[string]domain.OrderBook)
// }

func (c *Coordinator) StartOrderBookStream() {
	c.logger.Debug("start orderbook stream")
	var wg sync.WaitGroup
	names := make([]core_config.ExchangeName, 0, len(c.orderBookRunners))
	for name := range c.orderBookRunners {
		names = append(names, name)
	}
	sort.Slice(names, func(i, j int) bool { return names[i] < names[j] })
	for _, name := range names {
		runner := c.orderBookRunners[name]
		name, runner := name, runner
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := runner.Start(c.ctx); err != nil {
				c.logger.Warn(
					"order book stream unavailable",
					slog.String("exchange", string(name)),
					slog.Any("error", err),
				)
			}
		}()
	}
	wg.Wait()
}

func (c *Coordinator) UpdateOrderBookStream() {
	c.logger.Debug("update orderbook stream")
	result := c.matcherStore.Get()
	if result == nil {
		return
	}

	names := make([]core_config.ExchangeName, 0, len(c.orderBookRunners))
	for name := range c.orderBookRunners {
		names = append(names, name)
	}
	sort.Slice(names, func(i, j int) bool { return names[i] < names[j] })
	for _, name := range names {
		runner := c.orderBookRunners[name]
		if err := runner.UpdateCoins(c.ctx, result.Spot[name], result.Futures[name]); err != nil {
			c.logger.Warn(
				"could not update order book subscriptions",
				slog.String("exchange", string(name)),
				slog.Any("error", err),
			)
		}
	}
}

func (c *Coordinator) OrderBookSnapshot() OrderBookSnapshot {
	return c.orderBooks.Snapshot()
}

func (c *Coordinator) publishOrderBookSnapshot() {
	now := time.Now().UTC()
	snapshot := buildMarketSnapshot(c.matcherStore.Get(), c.OrderBookSnapshot(), now)
	size, err := c.snapshots.Publish(c.ctx, snapshot)
	if err != nil {
		c.logger.Warn("could not publish market snapshot", slog.Any("error", err))
		return
	}
	c.logger.Debug(
		"published market snapshot",
		slog.String("snapshot_id", snapshot.SnapshotID),
		slog.Int("bytes", size),
		slog.Int("matches", len(snapshot.Matches)),
	)
}
