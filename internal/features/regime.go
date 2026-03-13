package features

import "db_trace/kucoin-ising-bot/internal/types"

func DetectRegime(f types.FeatureState) types.RegimeState {
	if f.Mid <= 0 || f.BestBid <= 0 || f.BestAsk <= 0 {
		return types.RegimeState{Tradable: false, Reason: "bad_mid_or_book"}
	}

	if f.BestBid >= f.BestAsk {
		return types.RegimeState{Tradable: false, Reason: "crossed_book"}
	}

	spreadNorm := f.Spread / f.Mid
	if spreadNorm > 0.00008 {
		return types.RegimeState{Tradable: false, Reason: "spread_too_wide"}
	}
	if f.Vol10s < 0.000002 {
		return types.RegimeState{Tradable: false, Reason: "vol_too_low"}
	}
	if f.Vol10s > 0.0008 {
		return types.RegimeState{Tradable: false, Reason: "vol_too_high"}
	}
	if f.Entropy10s > 0.985 {
		return types.RegimeState{Tradable: false, Reason: "entropy_too_high"}
	}
	if f.IsingConsensus < 0.15 && f.IsingSuscept > 1.6 {
		return types.RegimeState{Tradable: false, Reason: "ising_critical_zone"}
	}
	if f.IsingCriticalness > 1.25 {
		return types.RegimeState{Tradable: false, Reason: "ising_transition_noise"}
	}

	return types.RegimeState{Tradable: true}
}
