package kucoin

import (
	"db_trace/kucoin-ising-bot/internal/types"
	"sort"
	"strconv"
	"sync"
	"time"
)

type LocalBook struct {
	mu sync.RWMutex

	symbol   string
	sequence int64

	bids map[float64]float64
	asks map[float64]float64
}

func NewLocalBook(symbol string) *LocalBook {
	return &LocalBook{
		symbol: symbol,
		bids:   make(map[float64]float64, 4096),
		asks:   make(map[float64]float64, 4096),
	}
}

func (b *LocalBook) LoadSnapshot(seq int64, bids, asks [][]string) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.sequence = seq
	clear(b.bids)
	clear(b.asks)

	load := func(dst map[float64]float64, rows [][]string) {
		for _, row := range rows {
			if len(row) < 2 {
				continue
			}
			p, err1 := strconv.ParseFloat(row[0], 64)
			q, err2 := strconv.ParseFloat(row[1], 64)
			if err1 != nil || err2 != nil {
				continue
			}
			if q > 0 {
				dst[p] = q
			}
		}
	}

	load(b.bids, bids)
	load(b.asks, asks)

	return nil
}

func (b *LocalBook) Sequence() int64 {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.sequence
}

func (b *LocalBook) ApplyDelta(sequenceStart, sequenceEnd int64, bidChanges, askChanges [][]string) bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	if !(sequenceStart <= b.sequence+1 && sequenceEnd > b.sequence) {
		return false
	}

	applySide := func(dst map[float64]float64, changes [][]string) {
		for _, row := range changes {
			if len(row) < 2 {
				continue
			}
			price, err1 := strconv.ParseFloat(row[0], 64)
			qty, err2 := strconv.ParseFloat(row[1], 64)
			if err1 != nil || err2 != nil {
				continue
			}
			if qty == 0 {
				delete(dst, price)
			} else {
				dst[price] = qty
			}
		}
	}

	applySide(b.bids, bidChanges)
	applySide(b.asks, askChanges)
	b.sequence = sequenceEnd
	return true
}

func (b *LocalBook) Snapshot(topN int) types.BookSnapshot {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if topN <= 0 {
		topN = 20
	}

	bidPrices := make([]float64, 0, len(b.bids))
	for p := range b.bids {
		bidPrices = append(bidPrices, p)
	}
	sort.Slice(bidPrices, func(i, j int) bool { return bidPrices[i] > bidPrices[j] })

	askPrices := make([]float64, 0, len(b.asks))
	for p := range b.asks {
		askPrices = append(askPrices, p)
	}
	sort.Slice(askPrices, func(i, j int) bool { return askPrices[i] < askPrices[j] })

	bids := make([]types.Level, 0, min(topN, len(bidPrices)))
	for i := 0; i < len(bidPrices) && i < topN; i++ {
		p := bidPrices[i]
		bids = append(bids, types.Level{Price: p, Qty: b.bids[p]})
	}

	asks := make([]types.Level, 0, min(topN, len(askPrices)))
	for i := 0; i < len(askPrices) && i < topN; i++ {
		p := askPrices[i]
		asks = append(asks, types.Level{Price: p, Qty: b.asks[p]})
	}

	return types.BookSnapshot{
		Symbol:    b.symbol,
		Bids:      bids,
		Asks:      asks,
		Timestamp: time.Now(),
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
