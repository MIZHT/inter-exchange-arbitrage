package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	core_config "github.com/MIZHT/inter-exchange-arbitrage/internal/core/config"
	core_logger "github.com/MIZHT/inter-exchange-arbitrage/internal/core/logger"
	"github.com/MIZHT/inter-exchange-arbitrage/internal/core/transport/websocket_manager"
	"github.com/MIZHT/inter-exchange-arbitrage/internal/engine"
	"github.com/MIZHT/inter-exchange-arbitrage/internal/exchange"
	"github.com/MIZHT/inter-exchange-arbitrage/internal/exchange/binance"
	"github.com/MIZHT/inter-exchange-arbitrage/internal/exchange/bingx"
	"github.com/MIZHT/inter-exchange-arbitrage/internal/exchange/bitget"
	"github.com/MIZHT/inter-exchange-arbitrage/internal/exchange/bitmart"
	"github.com/MIZHT/inter-exchange-arbitrage/internal/exchange/bybit"
	"github.com/MIZHT/inter-exchange-arbitrage/internal/exchange/coinex"
	"github.com/MIZHT/inter-exchange-arbitrage/internal/exchange/gateio"
	"github.com/MIZHT/inter-exchange-arbitrage/internal/exchange/kucoin"
	"github.com/MIZHT/inter-exchange-arbitrage/internal/exchange/mexc"
	"github.com/MIZHT/inter-exchange-arbitrage/internal/exchange/okx"
	"github.com/MIZHT/inter-exchange-arbitrage/internal/exchange/xt"
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
	proxyPool := websocket_manager.NewProxyPool(cfg.Proxies)
	proxy := ""
	if len(cfg.Proxies) > 0 {
		proxy = cfg.Proxies[0]
	}
	for name, exCfg := range cfg.Exchanges {
		switch name {
		case core_config.Binance:
			adapters[name] = binance.NewBinance(logger, exCfg, proxy, proxyPool)
		case core_config.Bitget:
			adapters[name] = bitget.NewBitget(logger, exCfg, proxy, proxyPool)
		case core_config.Bitmart:
			adapters[name] = bitmart.NewBitmart(logger, exCfg, proxy, proxyPool)
		case core_config.Bingx:
			adapters[name] = bingx.NewBingx(logger, exCfg, proxy, proxyPool)
		case core_config.Bybit:
			adapters[name] = bybit.NewBybit(logger, exCfg, proxy, proxyPool)
		case core_config.Coinex:
			adapters[name] = coinex.NewCoinEx(logger, exCfg, proxy, proxyPool)
		case core_config.Gateio:
			adapters[name] = gateio.NewGateIO(logger, exCfg, proxy, proxyPool)
		case core_config.Kucoin:
			adapters[name] = kucoin.NewKuCoin(logger, exCfg, proxy, proxyPool)
		case core_config.Mexc:
			adapters[name] = mexc.NewMEXC(logger, exCfg, proxy, proxyPool)
		case core_config.Okx:
			adapters[name] = okx.NewOKX(logger, exCfg, proxy, proxyPool)
		case core_config.Xt:
			adapters[name] = xt.NewXT(logger, exCfg, proxy, proxyPool)
		default:
			logger.Warn("no adapter for exchange", slog.String("exchange", string(name)))
		}
	}

	snapshotPublisher, err := engine.NewSnapshotPublisher(os.Getenv("MARKET_SNAPSHOT_URL"))
	if err != nil {
		logger.Error("Invalid market snapshot publisher configuration", slog.Any("error", err))
		return
	}
	coordinator := engine.NewCoordinator(ctx, logger, adapters, snapshotPublisher)
	if err := coordinator.Run(ctx); err != nil {
		logger.Error("Parser run error", slog.Any("error", err))
	}
}
