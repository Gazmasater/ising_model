package features

import (
	"db_trace/kucoin-ising-bot/internal/ring"
	"db_trace/kucoin-ising-bot/internal/types"
	"math"
	"time"
)

type Engine struct {
	Symbol string

	bestBid float64
	bestAsk float64
	bid1Qty float64
	ask1Qty float64

	mid    float64
	micro  float64
	spread float64
	tick   float64

	prevBestBid float64
	prevBestAsk float64
	prevBid1Qty float64
	prevAsk1Qty float64

	prevBidTop3 float64
	prevAskTop3 float64
	currBidTop3 float64
	currAskTop3 float64

	prevBidTop5 float64
	prevAskTop5 float64
	currBidTop5 float64
	currAskTop5 float64

	bookOFIWindow *ring.FloatRing
	driftWindow   *ring.FloatRing
	returnWindow  *ring.FloatRing

	tradeBucketStart time.Time
	tradeBucketValue float64
	tradeOFIWindow   *ring.FloatRing
	tradeBucketDur   time.Duration

	ising *IsingModel

	initialized bool
}

func NewEngine(symbol string, tradeBucket time.Duration, ising *IsingModel) *Engine {
	if tradeBucket <= 0 {
		tradeBucket = 50 * time.Millisecond
	}
	if ising == nil {
		ising = NewIsingModel(48, 2.2, 0.85, 1.0)
	}
	return &Engine{
		Symbol:         symbol,
		bookOFIWindow:  ring.NewFloatRing(64),
		driftWindow:    ring.NewFloatRing(64),
		returnWindow:   ring.NewFloatRing(334),
		tradeOFIWindow: ring.NewFloatRing(200),
		tradeBucketDur: tradeBucket,
		ising:          ising,
	}
}

func (e *Engine) UpdateBook(book types.BookSnapshot) {
	if len(book.Bids) == 0 || len(book.Asks) == 0 {
		return
	}

	bestBid := book.Bids[0].Price
	bestAsk := book.Asks[0].Price
	bid1Qty := book.Bids[0].Qty
	ask1Qty := book.Asks[0].Qty

	if bestBid <= 0 || bestAsk <= 0 || bestBid >= bestAsk || bid1Qty <= 0 || ask1Qty <= 0 {
		return
	}

	mid := 0.5 * (bestBid + bestAsk)
	spread := bestAsk - bestBid
	micro := computeMicro(bestBid, bestAsk, bid1Qty, ask1Qty)

	bidTop3 := sumTopNQty(book.Bids, 3)
	askTop3 := sumTopNQty(book.Asks, 3)
	bidTop5 := sumTopNQty(book.Bids, 5)
	askTop5 := sumTopNQty(book.Asks, 5)

	if !e.initialized {
		e.bestBid, e.bestAsk = bestBid, bestAsk
		e.bid1Qty, e.ask1Qty = bid1Qty, ask1Qty
		e.mid, e.micro, e.spread = mid, micro, spread
		e.tick = estimateTick(book.Bids, book.Asks)

		e.prevBestBid, e.prevBestAsk = bestBid, bestAsk
		e.prevBid1Qty, e.prevAsk1Qty = bid1Qty, ask1Qty

		e.prevBidTop3, e.prevAskTop3 = bidTop3, askTop3
		e.currBidTop3, e.currAskTop3 = bidTop3, askTop3

		e.prevBidTop5, e.prevAskTop5 = bidTop5, askTop5
		e.currBidTop5, e.currAskTop5 = bidTop5, askTop5

		e.bookOFIWindow.Add(0)
		e.driftWindow.Add(0)
		e.returnWindow.Add(0)
		e.tradeOFIWindow.Add(0)

		e.initialized = true
		return
	}

	prevMid := e.mid

	e.prevBidTop3, e.prevAskTop3 = e.currBidTop3, e.currAskTop3
	e.currBidTop3, e.currAskTop3 = bidTop3, askTop3

	e.prevBidTop5, e.prevAskTop5 = e.currBidTop5, e.currAskTop5
	e.currBidTop5, e.currAskTop5 = bidTop5, askTop5

	bookOFI := calcTopLevelOFI(
		e.prevBestBid, e.prevBid1Qty,
		e.prevBestAsk, e.prevAsk1Qty,
		bestBid, bid1Qty,
		bestAsk, ask1Qty,
	)
	e.bookOFIWindow.Add(bookOFI)
	e.driftWindow.Add(micro - e.micro)

	ret := 0.0
	if prevMid > 0 {
		ret = (mid - prevMid) / prevMid
	}
	e.returnWindow.Add(ret)

	e.bestBid, e.bestAsk = bestBid, bestAsk
	e.bid1Qty, e.ask1Qty = bid1Qty, ask1Qty
	e.mid, e.micro, e.spread = mid, micro, spread

	if e.tick == 0 {
		e.tick = estimateTick(book.Bids, book.Asks)
	}

	e.prevBestBid, e.prevBestAsk = bestBid, bestAsk
	e.prevBid1Qty, e.prevAsk1Qty = bid1Qty, ask1Qty
}

func (e *Engine) UpdateTrade(tr types.TradeTick) {
	if !e.initialized {
		return
	}

	ts := tr.Timestamp
	if ts.IsZero() {
		ts = time.Now()
	}

	if e.tradeBucketStart.IsZero() {
		e.tradeBucketStart = ts
	}

	for ts.Sub(e.tradeBucketStart) >= e.tradeBucketDur {
		e.tradeOFIWindow.Add(e.tradeBucketValue)
		e.tradeBucketValue = 0
		e.tradeBucketStart = e.tradeBucketStart.Add(e.tradeBucketDur)
	}

	switch tr.Side {
	case types.SideBuy:
		e.tradeBucketValue += tr.Qty
	case types.SideSell:
		e.tradeBucketValue -= tr.Qty
	}
}

func (e *Engine) ForceFlushTradeBucket(now time.Time) {
	if e.tradeBucketStart.IsZero() {
		e.tradeBucketStart = now
		return
	}
	for now.Sub(e.tradeBucketStart) >= e.tradeBucketDur {
		e.tradeOFIWindow.Add(e.tradeBucketValue)
		e.tradeBucketValue = 0
		e.tradeBucketStart = e.tradeBucketStart.Add(e.tradeBucketDur)
	}
}

func (e *Engine) Ready() bool {
	return e.initialized &&
		e.bookOFIWindow.Len() >= 16 &&
		e.driftWindow.Len() >= 16 &&
		e.returnWindow.Len() >= 64 &&
		e.tradeOFIWindow.Len() >= 16
}

func (e *Engine) Snapshot(now time.Time) types.FeatureState {
	e.ForceFlushTradeBucket(now)

	imb3 := computeImbalance(e.currBidTop3, e.currAskTop3)
	imb5 := computeImbalance(e.currBidTop5, e.currAskTop5)

	qdBid := computeDepletion(e.prevBidTop5, e.currBidTop5)
	qdAsk := computeDepletion(e.prevAskTop5, e.currAskTop5)
	qrBid := computeReplenishment(e.prevBidTop5, e.currBidTop5)
	qrAsk := computeReplenishment(e.prevAskTop5, e.currAskTop5)

	bookOFINorm := normSignedWindow(e.bookOFIWindow.Values())
	tradeOFINorm := normSignedWindow(e.tradeOFIWindow.Values())
	netOFI := clamp(0.55*bookOFINorm+0.45*tradeOFINorm, -1, 1)

	microDriftNorm := 0.0
	if last, ok := e.driftWindow.Last(); ok {
		den := e.mid
		if e.tick > 0 {
			den = math.Max(e.tick, e.mid*1e-6)
		}
		if den > 0 {
			microDriftNorm = last / den
		}
	}

	retVals := e.returnWindow.Values()
	ret1s := ring.Mean(lastN(retVals, 33))
	vol10s := ring.Std(lastN(retVals, 334))
	entropy := signEntropy(lastN(retVals, 334))

	f := types.FeatureState{
		Symbol:         e.Symbol,
		Mid:            e.mid,
		Micro:          e.micro,
		Spread:         e.spread,
		Tick:           e.tick,
		BestBid:        e.bestBid,
		BestAsk:        e.bestAsk,
		ImbalanceTop3:  imb3,
		ImbalanceTop5:  imb5,
		QDBid:          qdBid,
		QDAsk:          qdAsk,
		QRBid:          qrBid,
		QRAsk:          qrAsk,
		BookOFINorm:    bookOFINorm,
		TradeOFINorm:   tradeOFINorm,
		NetOFINorm:     netOFI,
		MicroDriftNorm: clamp(microDriftNorm, -1, 1),
		Return1s:       ret1s,
		Vol10s:         vol10s,
		Entropy10s:     entropy,
		UpdatedAt:      now,
	}

	if e.ising != nil {
		f = e.ising.Observe(f)
	}

	return f
}

func normSignedWindow(xs []float64) float64 {
	if len(xs) == 0 {
		return 0
	}
	den := ring.SumAbs(xs)
	if den == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range xs {
		sum += v
	}
	return clamp(sum/den, -1, 1)
}

func computeMicro(bid, ask, bidQty, askQty float64) float64 {
	den := bidQty + askQty
	if den == 0 {
		return 0.5 * (bid + ask)
	}
	return (ask*bidQty + bid*askQty) / den
}

func sumTopNQty(levels []types.Level, n int) float64 {
	sum := 0.0
	for i := 0; i < n && i < len(levels); i++ {
		sum += levels[i].Qty
	}
	return sum
}

func computeImbalance(bid, ask float64) float64 {
	den := bid + ask
	if den == 0 {
		return 0
	}
	return (bid - ask) / den
}

func computeDepletion(prev, curr float64) float64 {
	if prev <= 0 || curr >= prev {
		return 0
	}
	return (prev - curr) / prev
}

func computeReplenishment(prev, curr float64) float64 {
	if prev <= 0 || curr <= prev {
		return 0
	}
	return (curr - prev) / prev
}

func calcTopLevelOFI(prevBidPrice, prevBidQty, prevAskPrice, prevAskQty, bidPrice, bidQty, askPrice, askQty float64) float64 {
	var e float64

	switch {
	case bidPrice > prevBidPrice:
		e += bidQty
	case bidPrice == prevBidPrice:
		e += bidQty - prevBidQty
	case bidPrice < prevBidPrice:
		e -= prevBidQty
	}

	switch {
	case askPrice < prevAskPrice:
		e += askQty
	case askPrice == prevAskPrice:
		e -= (askQty - prevAskQty)
	case askPrice > prevAskPrice:
		e -= prevAskQty
	}

	return e
}

func estimateTick(bids, asks []types.Level) float64 {
	best := math.MaxFloat64

	for i := 1; i < len(bids); i++ {
		d := math.Abs(bids[i-1].Price - bids[i].Price)
		if d > 0 && d < best {
			best = d
		}
	}
	for i := 1; i < len(asks); i++ {
		d := math.Abs(asks[i-1].Price - asks[i].Price)
		if d > 0 && d < best {
			best = d
		}
	}

	if best == math.MaxFloat64 {
		return 0
	}
	return best
}

func signEntropy(xs []float64) float64 {
	if len(xs) == 0 {
		return 1
	}

	var pos, neg int
	for _, x := range xs {
		if x > 0 {
			pos++
		} else if x < 0 {
			neg++
		}
	}

	total := float64(pos + neg)
	if total == 0 {
		return 1
	}

	pp := float64(pos) / total
	pn := float64(neg) / total

	ent := 0.0
	if pp > 0 {
		ent -= pp * math.Log(pp)
	}
	if pn > 0 {
		ent -= pn * math.Log(pn)
	}

	return ent / math.Log(2)
}

func lastN(xs []float64, n int) []float64 {
	if len(xs) <= n {
		return xs
	}
	return xs[len(xs)-n:]
}

func clamp(x, lo, hi float64) float64 {
	if x < lo {
		return lo
	}
	if x > hi {
		return hi
	}
	return x
}
