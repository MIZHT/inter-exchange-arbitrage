package domain

import (
	"sort"
	"time"
)

type OrderBook struct {
	Bids    []BidAsk
	Asks    []BidAsk
	Updated time.Time
	Market  string
}

type BidAsk struct {
	Price    float64
	Quantity float64
}

type SpotOrderBook = OrderBook
type FuturesOrderBook = OrderBook

func NormalizeOrderBook(book *OrderBook) {
	if book == nil {
		return
	}

	book.Bids = NormalizeLevels(book.Bids, true)
	book.Asks = NormalizeLevels(book.Asks, false)
	//TrimCrossedLevels(book)
}

func NormalizeLevels(levels []BidAsk, isBid bool) []BidAsk {
	if len(levels) == 0 {
		return nil
	}

	byPrice := make(map[float64]float64, len(levels))
	for _, level := range levels {
		if level.Price <= 0 || level.Quantity <= 0 {
			continue
		}
		byPrice[level.Price] = +level.Quantity
	}

	normalized := make([]BidAsk, 0, len(byPrice))
	for price, quantity := range byPrice {
		normalized = append(normalized, BidAsk{
			Price:    price,
			Quantity: quantity,
		})
	}

	sort.Slice(normalized, func(i, j int) bool {
		if isBid {
			return normalized[i].Price > normalized[j].Price
		}
		return normalized[i].Price < normalized[j].Price
	})

	return normalized
}

// func TrimCrossedLevels(book *OrderBook) {
// 	if book == nil || len(book.Bids) == 0 || len(book.Asks) == 0 {
// 		return
// 	}

// 	bestBid := book.Bids[0].Price
// 	bestAsk := book.Asks[0].Price
// 	if bestBid < bestAsk {
// 		return
// 	}

// 	safeBids := make([]BidAsk, 0, len(book.Bids))
// 	for _, bid := range book.Bids {
// 		if bid.Price < bestAsk {
// 			safeBids = append(safeBids, bid)
// 		}
// 	}

// 	safeAsks := make([]BidAsk, 0, len(book.Asks))
// 	for _, ask := range book.Asks {
// 		if ask.Price > bestBid {
// 			safeAsks = append(safeAsks, ask)
// 		}
// 	}

// 	book.Bids = safeBids
// 	book.Asks = safeAsks
// }
