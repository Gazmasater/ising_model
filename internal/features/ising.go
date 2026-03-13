package features

import (
	"db_trace/kucoin-ising-bot/internal/ring"
	"db_trace/kucoin-ising-bot/internal/types"
	"math"
)

type IsingModel struct {
	beta       float64
	couplingJ  float64
	fieldScale float64

	spinWindow *ring.FloatRing

	initialized bool
	lastSpin    int
}

func NewIsingModel(window int, beta, couplingJ, fieldScale float64) *IsingModel {
	if window <= 0 {
		window = 48
	}
	if beta <= 0 {
		beta = 2.2
	}
	if fieldScale <= 0 {
		fieldScale = 1.0
	}
	return &IsingModel{
		beta:       beta,
		couplingJ:  couplingJ,
		fieldScale: fieldScale,
		spinWindow: ring.NewFloatRing(window),
	}
}

func (m *IsingModel) Observe(f types.FeatureState) types.FeatureState {
	oldMagnet := m.magnetization()
	h := clamp(m.externalField(f)*m.fieldScale, -2.0, 2.0)
	heff := m.couplingJ*oldMagnet + h

	pUp := sigmoid(2.0 * m.beta * heff)
	pDown := 1.0 - pUp

	spin := -1
	if pUp >= 0.5 {
		spin = 1
	}

	m.lastSpin = spin
	m.spinWindow.Add(float64(spin))
	m.initialized = true

	newMagnet := m.magnetization()
	chi := m.beta * (1.0 - newMagnet*newMagnet)
	if chi < 0 {
		chi = 0
	}

	energy := -float64(spin) * heff
	criticalness := chi * (1.0 - math.Abs(newMagnet))
	criticalness = clamp(criticalness, 0, 10)

	f.IsingField = h
	f.IsingMagnet = clamp(newMagnet, -1, 1)
	f.IsingEnergy = energy
	f.IsingProbUp = clamp(pUp, 0, 1)
	f.IsingProbDown = clamp(pDown, 0, 1)
	f.IsingSuscept = chi
	f.IsingSpin = spin
	f.IsingConsensus = math.Abs(newMagnet)
	f.IsingCriticalness = criticalness

	return f
}

func (m *IsingModel) externalField(f types.FeatureState) float64 {
	queueAsym := (f.QDAsk - f.QDBid) + 0.5*(f.QRBid-f.QRAsk)

	h :=
		0.24*f.BookOFINorm +
			0.20*f.TradeOFINorm +
			0.16*f.MicroDriftNorm +
			0.14*f.ImbalanceTop5 +
			0.10*f.ImbalanceTop3 +
			0.12*queueAsym +
			0.04*sign(f.Return1s)

	if f.Mid > 0 {
		spreadNorm := f.Spread / f.Mid
		h -= 18.0 * spreadNorm
	}

	return h
}

func (m *IsingModel) magnetization() float64 {
	xs := m.spinWindow.Values()
	if len(xs) == 0 {
		return 0
	}
	return clamp(ring.Mean(xs), -1, 1)
}

func sigmoid(x float64) float64 {
	if x > 40 {
		return 1
	}
	if x < -40 {
		return 0
	}
	return 1.0 / (1.0 + math.Exp(-x))
}

func sign(x float64) float64 {
	switch {
	case x > 0:
		return 1
	case x < 0:
		return -1
	default:
		return 0
	}
}
