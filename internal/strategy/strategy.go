package strategy

import (
	"db_trace/kucoin-ising-bot/internal/config"
	"db_trace/kucoin-ising-bot/internal/types"
	"time"
)

type Strategy struct {
	Pos           types.Position
	CooldownUntil time.Time

	MinHold           time.Duration
	MaxHold           time.Duration
	EmergencyStopFrac float64
	TakeProfitFrac    float64
	Cooldown          time.Duration
	Size              float64
}

func New(cfg config.Config) *Strategy {
	return &Strategy{
		MinHold:           cfg.MinHold,
		MaxHold:           cfg.MaxHold,
		EmergencyStopFrac: cfg.EmergencyStopFrac,
		TakeProfitFrac:    cfg.TakeProfitFrac,
		Cooldown:          cfg.Cooldown,
		Size:              cfg.PositionSize,
	}
}

func (s *Strategy) HasPosition() bool {
	return s.Pos.Side != types.SideUnknown
}

func (s *Strategy) CanEnter(now time.Time) bool {
	return !s.HasPosition() && now.After(s.CooldownUntil)
}

func (s *Strategy) Enter(sig types.Signal, f types.FeatureState, now time.Time) {
	if sig.Long {
		s.Pos = types.Position{
			Side:      types.SideBuy,
			Entry:     f.BestAsk,
			EntryBid:  f.BestBid,
			EntryAsk:  f.BestAsk,
			EntryTime: now,
			Size:      s.Size,
		}
		return
	}

	if sig.Short {
		s.Pos = types.Position{
			Side:      types.SideSell,
			Entry:     f.BestBid,
			EntryBid:  f.BestBid,
			EntryAsk:  f.BestAsk,
			EntryTime: now,
			Size:      s.Size,
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
			if f.IsingProbUp < 0.52 || f.IsingMagnet < 0.02 || f.IsingField < 0 {
				return true, "ising_reversal_long"
			}
			if f.NetOFINorm < 0 || f.Micro <= f.Mid || sig.LongScore < 0.14 {
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
			if f.IsingProbDown < 0.52 || f.IsingMagnet > -0.02 || f.IsingField > 0 {
				return true, "ising_reversal_short"
			}
			if f.NetOFINorm > 0 || f.Micro >= f.Mid || sig.ShortScore < 0.14 {
				return true, "model_exit_short"
			}
		}
	}

	return false, ""
}
