package cmc

import (
	"fmt"
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
	spot := make(map[string]struct{}, 3)
	futures := make(map[string]struct{}, 3)

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

func getName(id int) string {
	switch id {
	case 270:
		return "binance"
	case 311:
		return "kucoin"
	case 544:
		return "mexc"
	case 302:
		return "gateio"
	case 521:
		return "bybit"
	case 294:
		return "okx"
	case 513:
		return "bitget"
	case 1064:
		return "bingx"
	case 525:
		return "xt"
	default:
		return ""
	}
}

func groupSet(values map[string]struct{}) [][]string {
	if len(values) == 0 {
		return nil
	}

	group := make([]string, 0, len(values))
	for value := range values {
		group = append(group, value)
	}
	return [][]string{group}
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
