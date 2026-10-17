package cmc

import (
	"fmt"

	core_config "github.com/MIZHT/inter-exchange-arbitrage/internal/core/config"
)

func listIndex(index CmcIndexResponse) ([]int, error) {
	ids := make([]int, 0, len(index.Values))
	var skipped int

	for _, raw := range index.Values {
		if len(raw) < 3 {
			skipped++
			continue
		}

		f, ok := raw[0].(float64)
		if !ok {
			skipped++
			continue
		}
		ids = append(ids, int(f))
	}
	if skipped > 0 {
		return ids, fmt.Errorf("skipped %d malformed entries out of %d", skipped, len(index.Values))
	}
	return ids, nil
}

func MapTicker(
	coinID int,
	pairs []MarketPair,
) TickerOutput {
	spot := make(map[core_config.ExchangeName]struct{}, 3)
	futures := make(map[core_config.ExchangeName]struct{}, 3)

	for _, pair := range pairs {
		exchangeName := getName(pair.ExchangeID)
		if exchangeName == "" {
			continue
		}

		switch pair.Category {
		case "spot":
			spot[exchangeName] = struct{}{}
		case "perpetual", "futures":
			futures[exchangeName] = struct{}{}
		}
	}

	return TickerOutput{
		CoinID:  coinID,
		Spot:    groupSet(spot),
		Futures: groupSet(futures),
	}
}

func getName(id int) core_config.ExchangeName {
	switch id {
	case 270:
		return core_config.Binance
	case 311:
		return core_config.Kucoin
	case 544:
		return core_config.Mexc
	case 302:
		return core_config.Gateio
	case 521:
		return core_config.Bybit
	case 294:
		return core_config.Okx
	case 513:
		return core_config.Bitget
	case 1064:
		return core_config.Bingx
	case 525:
		return core_config.Xt
	default:
		return ""
	}
}

func groupSet(values map[core_config.ExchangeName]struct{}) [][]core_config.ExchangeName {
	if len(values) == 0 {
		return nil
	}

	group := make([]core_config.ExchangeName, 0, len(values))
	for value := range values {
		group = append(group, value)
	}
	return [][]core_config.ExchangeName{group}
}

func makeSnapshot(symbol string, mapTicker TickerOutput, Snapshot Snapshot) {
	if _, ok := Snapshot[symbol]; !ok {
		Snapshot[symbol] = mapTicker
	} else {
		item := Snapshot[symbol]
		item.Spot = append(item.Spot, mapTicker.Spot...)
		item.Futures = append(item.Futures, mapTicker.Futures...)
		Snapshot[symbol] = item
	}
}
