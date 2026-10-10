package domain

import (
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
}

func NormalizeLevels(levels []BidAsk, isBid bool) []BidAsk {
	n := len(levels)
	if n == 0 {
		return nil
	}

	reverse := false

	if n > 3 {
		ascending := levels[0].Price < levels[n-1].Price
		// Bids: по убыванию.
		// Asks: по возрастанию.
		reverse = (isBid && ascending) || (!isBid && !ascending)
	}

	if reverse {
		for i, j := 0, n-1; i < j; i, j = i+1, j-1 {
			levels[i], levels[j] = levels[j], levels[i]
		}
	}

	write := 0

	for read := 0; read < n; read++ {
		level := levels[read]

		if level.Price <= 0 || level.Quantity <= 0 {
			continue
		}

		if write > 0 && levels[write-1].Price == level.Price {
			levels[write-1].Quantity += level.Quantity
			continue
		}

		levels[write] = level
		write++
	}
	if write == 0 {
		return nil
	}
	return levels[:write]
}
