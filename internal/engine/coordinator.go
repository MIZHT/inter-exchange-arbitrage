package engine

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/MIZHT/inter-exchange-arbitrage/internal/cmc"
	core_config "github.com/MIZHT/inter-exchange-arbitrage/internal/core/config"
	core_logger "github.com/MIZHT/inter-exchange-arbitrage/internal/core/logger"
	"github.com/MIZHT/inter-exchange-arbitrage/internal/exchange"
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

	c := &Coordinator{
		ctx:          ctx,
		logger:       logger,
		runners:      make(map[core_config.ExchangeName]*ExchangeRunner),
		cmcStore:     cmcStore,
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
				c.CMCParse(c.ctx)

			}()
		}
	}
}

func (c *Coordinator) ExFetch(ctx context.Context) {
	c.logger.Debug("start fetch exchanges")

	var wg sync.WaitGroup
	for exchangeName, runner := range c.runners {
		runner := runner
		exchangeName := exchangeName
		c.logger.Debug("start exchange fetch", slog.String("exchange", string(exchangeName)))
		wg.Add(1)
		go func() {
			defer wg.Done()
			runner.RunExFetch(ctx)
		}()
	}
	wg.Wait()
}

func (c *Coordinator) CMCParse(ctx context.Context) {
	c.logger.Debug("start cmc")

	if err := cmc.RunCmc(ctx, c.logger, c.proxies, c.cmcStore); err != nil {
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
