package config

import "time"

type Config struct {
	Symbol string

	TopLevels int

	WSReadTimeout  time.Duration
	ReconnectDelay time.Duration

	BookThrottle time.Duration
	Warmup       time.Duration

	TradeBucket time.Duration

	SignalCSV string
	TradeCSV  string

	PositionSize float64
	TakerFeeRate float64
	SlippageFrac float64

	Cooldown          time.Duration
	MinHold           time.Duration
	MaxHold           time.Duration
	EmergencyStopFrac float64
	TakeProfitFrac    float64

	IsingWindow int
	IsingBeta   float64
	IsingJ      float64
	IsingScale  float64
}

func Load() Config {
	return Config{
		Symbol:         "BTC-USDT",
		TopLevels:      20,
		WSReadTimeout:  30 * time.Second,
		ReconnectDelay: 3 * time.Second,

		BookThrottle: 30 * time.Millisecond,
		Warmup:       8 * time.Second,

		TradeBucket: 50 * time.Millisecond,

		SignalCSV: "signals_kucoin_ising.csv",
		TradeCSV:  "trades_kucoin_paper.csv",

		PositionSize: 1.0,
		TakerFeeRate: 0.001,
		SlippageFrac: 0.00005,

		Cooldown:          350 * time.Millisecond,
		MinHold:           250 * time.Millisecond,
		MaxHold:           2500 * time.Millisecond,
		EmergencyStopFrac: 0.0006,
		TakeProfitFrac:    0.0007,

		IsingWindow: 48,
		IsingBeta:   2.2,
		IsingJ:      0.85,
		IsingScale:  1.0,
	}
}
