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

	// Микроструктура — главный источник edge.
	microLong :=
		0.16*f.QDAsk -
			0.08*f.QRAsk +
			0.18*f.BookOFINorm +
			0.20*f.TradeOFINorm +
			0.14*f.MicroDriftNorm +
			0.12*f.ImbalanceTop5 +
			0.08*f.ImbalanceTop3 +
			0.08*f.NetOFINorm

	microShort :=
		0.16*f.QDBid -
			0.08*f.QRBid +
			0.18*(-f.BookOFINorm) +
			0.20*(-f.TradeOFINorm) +
			0.14*(-f.MicroDriftNorm) +
			0.12*(-f.ImbalanceTop5) +
			0.08*(-f.ImbalanceTop3) +
			0.08*(-f.NetOFINorm)

	// Ising — только подтверждение, а не доминирующий сигнал.
	isingLong := 0.24*f.IsingProbUp + 0.08*f.IsingConsensus - 0.10*f.IsingCriticalness
	isingShort := 0.24*f.IsingProbDown + 0.08*f.IsingConsensus - 0.10*f.IsingCriticalness

	longScore := microLong + isingLong
	shortScore := microShort + isingShort

	s.LongScore = longScore
	s.ShortScore = shortScore

	long :=
		f.IsingProbUp > 0.80 &&
			f.IsingMagnet > 0.20 &&
			f.IsingField > 0.05 &&
			f.NetOFINorm > 0.20 &&
			f.Micro > f.Mid &&
			f.MicroDriftNorm > 0.05 &&
			f.ImbalanceTop5 > 0.10 &&
			longScore > 0.65

	short :=
		f.IsingProbDown > 0.80 &&
			f.IsingMagnet < -0.20 &&
			f.IsingField < -0.05 &&
			f.NetOFINorm < -0.20 &&
			f.Micro < f.Mid &&
			f.MicroDriftNorm < -0.05 &&
			f.ImbalanceTop5 < -0.10 &&
			shortScore > 0.65

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
