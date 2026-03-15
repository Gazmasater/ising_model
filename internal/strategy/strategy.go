package strategy

import (
	"time"

	"db_trace/kucoin-ising-bot/internal/config"
	"db_trace/kucoin-ising-bot/internal/types"
)

type Strategy struct {
	Pos           types.Position
	CooldownUntil time.Time

	MinHold           time.Duration
	MaxHold           time.Duration
	EmergencyStopFrac float64
	TakeProfitFrac    float64
	Cooldown          time.Duration

	NotionalUSDT       float64
	FuturesMode        bool
	ContractMultiplier float64
}

func New(cfg config.Config) *Strategy {
	notional := cfg.PositionNotionalUSDT
	if notional <= 0 {
		notional = 100.0
	}

	mult := cfg.ContractMultiplier
	if mult <= 0 {
		mult = 0.001
	}

	return &Strategy{
		MinHold:            cfg.MinHold,
		MaxHold:            cfg.MaxHold,
		EmergencyStopFrac:  cfg.EmergencyStopFrac,
		TakeProfitFrac:     cfg.TakeProfitFrac,
		Cooldown:           cfg.Cooldown,
		NotionalUSDT:       notional,
		FuturesMode:        cfg.FuturesMode,
		ContractMultiplier: mult,
	}
}

func (s *Strategy) HasPosition() bool {
	return s.Pos.Side != types.SideUnknown
}

func (s *Strategy) CanEnter(now time.Time) bool {
	return !s.HasPosition() && now.After(s.CooldownUntil)
}

func (s *Strategy) calcSize(price float64) float64 {
	if price <= 0 {
		return 0
	}

	if s.FuturesMode {
		contracts := s.NotionalUSDT / (price * s.ContractMultiplier)
		if contracts < 1 {
			contracts = 1
		}
		return contracts
	}

	return s.NotionalUSDT / price
}

func (s *Strategy) Enter(sig types.Signal, f types.FeatureState, now time.Time) {
	if sig.Long {
		if f.BestAsk <= 0 {
			return
		}

		size := s.calcSize(f.BestAsk)
		if size <= 0 {
			return
		}

		s.Pos = types.Position{
			Side:      types.SideBuy,
			Entry:     f.BestAsk,
			EntryBid:  f.BestBid,
			EntryAsk:  f.BestAsk,
			EntryTime: now,
			Size:      size,
		}
		return
	}

	if sig.Short {
		if f.BestBid <= 0 {
			return
		}

		size := s.calcSize(f.BestBid)
		if size <= 0 {
			return
		}

		s.Pos = types.Position{
			Side:      types.SideSell,
			Entry:     f.BestBid,
			EntryBid:  f.BestBid,
			EntryAsk:  f.BestAsk,
			EntryTime: now,
			Size:      size,
		}
	}
}

func (s *Strategy) Exit(now time.Time) {
	s.Pos = types.Position{}
	s.CooldownUntil = now.Add(s.Cooldown)
}

func (s *Strategy) ShouldExit(f types.FeatureState, sig types.Signal, now time.Time) (bool, string) {
	if !s.HasPosition() {
		return false, ""
	}

	held := now.Sub(s.Pos.EntryTime)
	if held > s.MaxHold {
		return true, "max_hold"
	}

	switch s.Pos.Side {
	case types.SideBuy:
		if f.BestBid <= 0 || s.Pos.Entry <= 0 {
			return false, ""
		}

		move := (f.BestBid - s.Pos.Entry) / s.Pos.Entry

		if move <= -s.EmergencyStopFrac {
			return true, "emergency_stop_long"
		}
		if move >= s.TakeProfitFrac {
			return true, "take_profit_long"
		}

		if held >= s.MinHold {
			// Более мягкий выход по развороту.
			if f.IsingProbUp < 0.45 || f.IsingMagnet < -0.05 || f.IsingField < -0.05 {
				return true, "ising_reversal_long"
			}

			// Выход только если реально модель сломалась.
			if f.NetOFINorm < -0.08 || f.Micro < f.Mid || sig.LongScore < 0.10 {
				return true, "model_exit_long"
			}
		}

	case types.SideSell:
		if f.BestAsk <= 0 || s.Pos.Entry <= 0 {
			return false, ""
		}

		move := (s.Pos.Entry - f.BestAsk) / s.Pos.Entry

		if move <= -s.EmergencyStopFrac {
			return true, "emergency_stop_short"
		}
		if move >= s.TakeProfitFrac {
			return true, "take_profit_short"
		}

		if held >= s.MinHold {
			if f.IsingProbDown < 0.45 || f.IsingMagnet > 0.05 || f.IsingField > 0.05 {
				return true, "ising_reversal_short"
			}

			if f.NetOFINorm > 0.08 || f.Micro > f.Mid || sig.ShortScore < 0.10 {
				return true, "model_exit_short"
			}
		}
	}

	return false, ""
}
