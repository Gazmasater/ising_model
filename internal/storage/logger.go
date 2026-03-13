package storage

import (
	"db_trace/kucoin-ising-bot/internal/types"
	"encoding/csv"
	"os"
	"strconv"
	"time"
)

type CSVLogger struct {
	f *os.File
	w *csv.Writer
}

func NewSignalLogger(path string) (*CSVLogger, error) {
	f, err := os.Create(path)
	if err != nil {
		return nil, err
	}
	w := csv.NewWriter(f)

	_ = w.Write([]string{
		"ts", "symbol",
		"best_bid", "best_ask", "mid", "micro", "spread", "tick",
		"imb3", "imb5",
		"qd_bid", "qd_ask", "qr_bid", "qr_ask",
		"book_ofi", "trade_ofi", "net_ofi",
		"micro_drift", "ret1s", "vol10s", "entropy10s",
		"ising_field", "ising_magnet", "ising_energy",
		"ising_prob_up", "ising_prob_down", "ising_suscept",
		"ising_spin", "ising_consensus", "ising_criticalness",
		"long_score", "short_score", "long", "short", "reason",
	})
	w.Flush()

	return &CSVLogger{f: f, w: w}, nil
}

func NewTradeLogger(path string) (*CSVLogger, error) {
	f, err := os.Create(path)
	if err != nil {
		return nil, err
	}
	w := csv.NewWriter(f)

	_ = w.Write([]string{
		"ts", "symbol", "event", "side", "price", "size",
		"entry", "exit", "gross_pnl", "net_pnl", "reason",
	})
	w.Flush()

	return &CSVLogger{f: f, w: w}, nil
}

func (l *CSVLogger) LogFeatureSignal(f types.FeatureState, s types.Signal) {
	_ = l.w.Write([]string{
		time.Now().Format(time.RFC3339Nano),
		f.Symbol,
		ff(f.BestBid), ff(f.BestAsk), ff(f.Mid), ff(f.Micro), ff(f.Spread), ff(f.Tick),
		ff(f.ImbalanceTop3), ff(f.ImbalanceTop5),
		ff(f.QDBid), ff(f.QDAsk), ff(f.QRBid), ff(f.QRAsk),
		ff(f.BookOFINorm), ff(f.TradeOFINorm), ff(f.NetOFINorm),
		ff(f.MicroDriftNorm), ff(f.Return1s), ff(f.Vol10s), ff(f.Entropy10s),
		ff(f.IsingField), ff(f.IsingMagnet), ff(f.IsingEnergy),
		ff(f.IsingProbUp), ff(f.IsingProbDown), ff(f.IsingSuscept),
		strconv.Itoa(f.IsingSpin), ff(f.IsingConsensus), ff(f.IsingCriticalness),
		ff(s.LongScore), ff(s.ShortScore),
		strconv.FormatBool(s.Long), strconv.FormatBool(s.Short), s.Reason,
	})
	l.w.Flush()
}

func (l *CSVLogger) LogTrade(
	symbol, event, side string,
	price, size, entry, exit, grossPnL, netPnL float64,
	reason string,
) {
	_ = l.w.Write([]string{
		time.Now().Format(time.RFC3339Nano),
		symbol, event, side,
		ff(price), ff(size), ff(entry), ff(exit),
		ff(grossPnL), ff(netPnL), reason,
	})
	l.w.Flush()
}

func (l *CSVLogger) Close() error {
	l.w.Flush()
	return l.f.Close()
}

func ff(v float64) string {
	return strconv.FormatFloat(v, 'f', 8, 64)
}
