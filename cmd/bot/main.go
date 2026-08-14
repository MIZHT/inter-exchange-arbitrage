package main

import (
	core_config "github.com/MIZHT/inter-exchange-arbitrage/internal/core/config"
	core_logger "github.com/MIZHT/inter-exchange-arbitrage/internal/core/logger"
)

func main() {
	cfg := core_config.NewConfigMust()
	logger := core_logger.NewLoggerMust(cfg.Logger)
	defer logger.Close()

}
