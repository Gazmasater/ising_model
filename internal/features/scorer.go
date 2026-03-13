package features

import "db_trace/kucoin-ising-bot/internal/types"

func Score(f types.FeatureState, r types.RegimeState) types.Signal {
	s := types.Signal{
		Symbol:    f.Symbol,
		NoTrade:   true,
		Reason:    r.Reason,
		CreatedAt: f.UpdatedAt,
		ProbLong:  f.IsingProbUp,
		ProbShort: f.IsingProbDown,
	}

	if !r.Tradable {
		return s
	}

	microLong :=
		0.18*f.QDAsk -
			0.10*f.QRAsk +
			0.16*f.BookOFINorm +
			0.18*f.TradeOFINorm +
			0.14*f.MicroDriftNorm +
			0.08*f.ImbalanceTop5 +
			0.06*f.ImbalanceTop3

	microShort :=
		0.18*f.QDBid -
			0.10*f.QRBid +
			0.16*(-f.BookOFINorm) +
			0.18*(-f.TradeOFINorm) +
			0.14*(-f.MicroDriftNorm) +
			0.08*(-f.ImbalanceTop5) +
			0.06*(-f.ImbalanceTop3)

	isingLong := 0.42*f.IsingProbUp + 0.18*f.IsingConsensus - 0.14*f.IsingCriticalness
	isingShort := 0.42*f.IsingProbDown + 0.18*f.IsingConsensus - 0.14*f.IsingCriticalness

	longScore := microLong + isingLong
	shortScore := microShort + isingShort

	s.LongScore = longScore
	s.ShortScore = shortScore

	long :=
		f.IsingProbUp > 0.64 &&
			f.IsingMagnet > 0.12 &&
			f.IsingField > 0 &&
			f.NetOFINorm > 0.12 &&
			f.Micro > f.Mid &&
			f.MicroDriftNorm > 0 &&
			f.ImbalanceTop5 > 0.05 &&
			longScore > 0.40

	short :=
		f.IsingProbDown > 0.64 &&
			f.IsingMagnet < -0.12 &&
			f.IsingField < 0 &&
			f.NetOFINorm < -0.12 &&
			f.Micro < f.Mid &&
			f.MicroDriftNorm < 0 &&
			f.ImbalanceTop5 < -0.05 &&
			shortScore > 0.40

	if long {
		s.Long = true
		s.NoTrade = false
		s.Reason = "ising_long_signal"
		return s
	}
	if short {
		s.Short = true
		s.NoTrade = false
		s.Reason = "ising_short_signal"
		return s
	}

	s.Reason = "no_strong_signal"
	return s
}
