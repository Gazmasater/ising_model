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

	// Размер входа в USDT notional.
	PositionNotionalUSDT float64

	// Комиссия taker на одну сторону.
	TakerFeeRate float64

	// Проскальзывание на одну сторону.
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

	FuturesMode        bool
	ContractMultiplier float64 // XBTUSDTM = 0.001 BTC per contract
}

func Load() Config {
	return Config{
		Symbol:    "XBTUSDTM",
		TopLevels: 5,

		WSReadTimeout:  30 * time.Second,
		ReconnectDelay: 3 * time.Second,

		BookThrottle: 30 * time.Millisecond,
		Warmup:       10 * time.Second,

		TradeBucket: 50 * time.Millisecond,

		SignalCSV: "signals_kucoin_futures_ising.csv",
		TradeCSV:  "trades_kucoin_futures_paper.csv",

		// Можно потом попробовать 1000 / 2000 / 3000 и сравнить.
		PositionNotionalUSDT: 3000.0,

		// Futures taker 0.06% со скидкой KCS 20%:
		// 0.0006 * 0.8 = 0.00048
		TakerFeeRate: 0.00048,

		// Небольшое paper-проскальзывание.
		SlippageFrac: 0.00002,

		// Более редкая торговля и более длинный hold,
		// чтобы брать ход, способный перекрыть fee.
		Cooldown:          2 * time.Second,
		MinHold:           1500 * time.Millisecond,
		MaxHold:           20 * time.Second,
		EmergencyStopFrac: 0.0015, // 0.15%
		TakeProfitFrac:    0.0040, // 0.40%

		// Ещё немного ослабленный Ising.
		IsingWindow: 24,
		IsingBeta:   0.90,
		IsingJ:      0.12,
		IsingScale:  0.45,

		FuturesMode:        true,
		ContractMultiplier: 0.001,
	}
}
