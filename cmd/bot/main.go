package main

import (
	"context"
	"os/signal"
	"syscall"

	core_config "github.com/MIZHT/inter-exchange-arbitrage/internal/core/config"
	core_logger "github.com/MIZHT/inter-exchange-arbitrage/internal/core/logger"
	"github.com/MIZHT/inter-exchange-arbitrage/internal/engine"
	"github.com/MIZHT/inter-exchange-arbitrage/internal/exchange"
	"github.com/MIZHT/inter-exchange-arbitrage/internal/exchange/bybit"
	"go.uber.org/zap"
)

func main() {
	cfg := core_config.NewConfigMust()

	logger := core_logger.NewLoggerMust(cfg.Logger)
	defer logger.Close()

	ctx, cancel := signal.NotifyContext(
		context.Background(),
		syscall.SIGINT, syscall.SIGTERM,
	)
	defer cancel()

	adapters := make(map[core_config.ExchangeName]exchange.ExchangeAdapter)
	for name, exCfg := range cfg.Exchanges {
		switch name {
		case core_config.Bybit:
			adapters[name] = bybit.NewBybit(logger, exCfg)

		default:
			logger.Warn("no adapter for exchange", zap.String("exchange", string(name)))
		}
	}

	coordinator := engine.NewCoordinator(ctx, logger, adapters)
	if err := coordinator.Run(ctx); err != nil {
		logger.Error("Parser run error", zap.Error(err))
	}
}
