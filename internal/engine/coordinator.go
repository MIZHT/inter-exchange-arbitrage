package engine

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/MIZHT/inter-exchange-arbitrage/internal/cmc"
	core_config "github.com/MIZHT/inter-exchange-arbitrage/internal/core/config"
	core_logger "github.com/MIZHT/inter-exchange-arbitrage/internal/core/logger"
	"github.com/MIZHT/inter-exchange-arbitrage/internal/exchange"
	"golang.org/x/sync/errgroup"
)

type Coordinator struct {
	mu      sync.Mutex
	ctx     context.Context
	cancel  context.CancelFunc
	logger  *core_logger.Logger
	proxies []string

	cmcStore     *cmc.Store
	matcherStore *MatchStore

	runners map[core_config.ExchangeName]*ExchangeRunner
}

func NewCoordinator(
	ctx context.Context,
	logger *core_logger.Logger,
	adapters map[core_config.ExchangeName]exchange.ExchangeAdapter,
) *Coordinator {
	c := &Coordinator{
		ctx:          ctx,
		logger:       logger,
		runners:      make(map[core_config.ExchangeName]*ExchangeRunner),
		cmcStore:     cmc.NewStore(),
		matcherStore: NewMatches(),
	}
	for name, ex := range adapters {
		c.runners[name] = NewExchangeRunner(ex, logger)

	}
	return c
}

type MatchInput struct {
	Exchanges map[core_config.ExchangeName]*ExchangeSnapshot
	CMC       *cmc.Snapshot
}

func (c *Coordinator) Run(ctx context.Context) error {
	var wg sync.WaitGroup
	c.ctx, c.cancel = context.WithCancel(ctx)

	tickerExFetch := time.NewTicker(60 * time.Second)
	defer tickerExFetch.Stop()
	wg.Add(1)
	go func() {
		defer wg.Done()
		c.ExFetch(c.ctx)

	}()

	tickerCMC := time.NewTicker(30 * time.Minute)
	defer tickerCMC.Stop()
	wg.Add(1)
	go func() {
		defer wg.Done()
		c.CMCParse(c.ctx)

	}()

	wg.Wait()

	c.MatchFind()

	for {
		select {
		case <-c.ctx.Done():
			return nil
		case <-tickerExFetch.C:
			wg.Add(1)
			go func() {
				defer wg.Done()
				c.ExFetch(c.ctx)
				c.MatchFind()

			}()
		case <-tickerCMC.C:
			wg.Add(1)
			go func() {
				defer wg.Done()
				go c.CMCParse(c.ctx)

			}()
		}
	}
}

func (c *Coordinator) ExFetch(ctx context.Context) error {
	c.logger.Debug("start fetch exchanges")

	var wg sync.WaitGroup
	g, ctx := errgroup.WithContext(ctx)
	for exchangeName, runner := range c.runners {
		c.logger.Info("Add exchange runner", slog.String("exchangeName:", string(exchangeName)))
		wg.Add(1)
		g.Go(func() error {
			defer wg.Done()
			if err := runner.RunExFetch(ctx); err != nil {
				return fmt.Errorf("fetch exchange %q: %w", exchangeName, err)
			}
			return nil
		})
	}
	return g.Wait()
}

func (c *Coordinator) CMCParse(ctx context.Context) error {
	c.logger.Debug("start cmc")

	if err := cmc.RunCmc(ctx, c.logger, c.proxies, c.cmcStore); err != nil {
		return fmt.Errorf("parse CMC data: %w", err)
	}
	return nil
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
