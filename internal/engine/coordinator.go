package engine

import (
	"context"
	"fmt"
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

	cmcStore *cmc.Store

	adapters map[core_config.ExchangeName]*ExchangeRunner
}

func NewCoordinator(
	ctx context.Context,
	logger *core_logger.Logger,
	adapters map[core_config.ExchangeName]exchange.ExchangeAdapter,
) *Coordinator {
	c := &Coordinator{
		ctx:      ctx,
		logger:   logger,
		adapters: make(map[core_config.ExchangeName]*ExchangeRunner),
		cmcStore: cmc.NewStore(),
	}
	for name, ex := range adapters {
		runner := NewExchangeRunner(ex, logger)
		c.adapters[name] = runner
	}
	return c
}

func (c *Coordinator) Run(ctx context.Context) error {
	c.ctx, c.cancel = context.WithCancel(ctx)
	tickerSync := time.NewTicker(60 * time.Second)
	defer tickerSync.Stop()
	c.SyncAll(ctx)

	tickerCMC := time.NewTicker(15 * time.Minute)
	defer tickerCMC.Stop()
	c.CMCParse(ctx)

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-tickerSync.C:
			c.SyncAll(ctx)
		case <-tickerCMC.C:
			c.CMCParse(ctx)
		}
	}
}

func (c *Coordinator) SyncAll(ctx context.Context) {

}

func (c *Coordinator) CMCParse(ctx context.Context) error {
	if err := cmc.RunCmc(c.ctx, c.logger, c.proxies, c.cmcStore); err != nil {
		return fmt.Errorf("dfdf: %w", err)
	}
	return nil
}
