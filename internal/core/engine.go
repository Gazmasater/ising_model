package core

import (
	"context"
	"db_trace/kucoin-ising-bot/internal/config"
	"db_trace/kucoin-ising-bot/internal/features"
	"db_trace/kucoin-ising-bot/internal/storage"
	"db_trace/kucoin-ising-bot/internal/strategy"
	"db_trace/kucoin-ising-bot/internal/types"
	"fmt"
	"time"
)

type Engine struct {
	cfg config.Config

	events chan Event

	features *features.Engine
	strategy *strategy.Strategy

	signalLog *storage.CSVLogger
	tradeLog  *storage.CSVLogger

	startedAt       time.Time
	lastBookProcess time.Time
	warmedUp        bool
}

func NewEngine(cfg config.Config) (*Engine, error) {
	signalLog, err := storage.NewSignalLogger(cfg.SignalCSV)
	if err != nil {
		return nil, err
	}

	tradeLog, err := storage.NewTradeLogger(cfg.TradeCSV)
	if err != nil {
		_ = signalLog.Close()
		return nil, err
	}

	ising := features.NewIsingModel(cfg.IsingWindow, cfg.IsingBeta, cfg.IsingJ, cfg.IsingScale)

	return &Engine{
		cfg:       cfg,
		events:    make(chan Event, 8192),
		features:  features.NewEngine(cfg.Symbol, cfg.TradeBucket, ising),
		strategy:  strategy.New(cfg),
		signalLog: signalLog,
		tradeLog:  tradeLog,
		startedAt: time.Now(),
	}, nil
}

func (e *Engine) Events() chan<- Event {
	return e.events
}

func (e *Engine) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case ev := <-e.events:
			switch ev.Type {
			case EventTrade:
				if ev.Trade != nil {
					e.onTrade(*ev.Trade)
				}
			case EventBook:
				if ev.Book != nil {
					e.onBook(*ev.Book)
				}
			}
		}
	}
}

func (e *Engine) onTrade(tr types.TradeTick) {
	e.features.UpdateTrade(tr)
}

func (e *Engine) onBook(book types.BookSnapshot) {
	now := time.Now()

	if !e.lastBookProcess.IsZero() && now.Sub(e.lastBookProcess) < e.cfg.BookThrottle {
		return
	}
	e.lastBookProcess = now

	e.features.UpdateBook(book)
	if !e.features.Ready() {
		return
	}

	f := e.features.Snapshot(now)
	if f.Mid <= 0 || len(book.Bids) == 0 || len(book.Asks) == 0 {
		return
	}

	if !e.warmedUp {
		if now.Sub(e.startedAt) < e.cfg.Warmup {
			return
		}
		e.warmedUp = true
	}

	r := features.DetectRegime(f)
	sig := features.Score(f, r)

	if e.signalLog != nil {
		e.signalLog.LogFeatureSignal(f, sig)
	}

	if e.strategy.CanEnter(now) {
		if sig.Long || sig.Short {
			e.strategy.Enter(sig, f, now)

			side := "long"
			price := f.BestAsk
			if sig.Short {
				side = "short"
				price = f.BestBid
			}

			if e.tradeLog != nil {
				e.tradeLog.LogTrade(
					f.Symbol, "enter", side,
					price, e.strategy.Pos.Size, price, 0, 0, 0, sig.Reason,
				)
			}

			fmt.Printf(
				"ENTER %s side=%s px=%.2f bid=%.2f ask=%.2f L=%.3f S=%.3f pUp=%.3f pDn=%.3f m=%.3f h=%.3f\n",
				sig.Reason, side, price, f.BestBid, f.BestAsk,
				sig.LongScore, sig.ShortScore, f.IsingProbUp, f.IsingProbDown, f.IsingMagnet, f.IsingField,
			)
		}
		return
	}

	if exit, reason := e.strategy.ShouldExit(f, sig, now); exit {
		side, exitPx, grossPnL, netPnL := e.closePnL(f)

		if e.tradeLog != nil {
			e.tradeLog.LogTrade(
				f.Symbol, "exit", side,
				exitPx, e.strategy.Pos.Size,
				e.strategy.Pos.Entry, exitPx,
				grossPnL, netPnL, reason,
			)
		}

		fmt.Printf(
			"EXIT %s side=%s exit=%.2f gross=%.6f net=%.6f pUp=%.3f pDn=%.3f m=%.3f h=%.3f\n",
			reason, side, exitPx, grossPnL, netPnL,
			f.IsingProbUp, f.IsingProbDown, f.IsingMagnet, f.IsingField,
		)

		e.strategy.Exit(now)
	}
}

func (e *Engine) closePnL(f types.FeatureState) (side string, exitPx, grossPnL, netPnL float64) {
	pos := e.strategy.Pos
	size := pos.Size

	switch pos.Side {
	case types.SideBuy:
		side = "long"
		exitPx = f.BestBid * (1.0 - e.cfg.SlippageFrac)
		grossPnL = (exitPx - pos.Entry) * size
	case types.SideSell:
		side = "short"
		exitPx = f.BestAsk * (1.0 + e.cfg.SlippageFrac)
		grossPnL = (pos.Entry - exitPx) * size
	default:
		return "", 0, 0, 0
	}

	turnover := (pos.Entry + exitPx) * size
	fees := turnover * e.cfg.TakerFeeRate
	netPnL = grossPnL - fees

	return side, exitPx, grossPnL, netPnL
}

func (e *Engine) Close() {
	if e.signalLog != nil {
		_ = e.signalLog.Close()
	}
	if e.tradeLog != nil {
		_ = e.tradeLog.Close()
	}
}
