package types

import "time"

type FeatureState struct {
	Symbol string

	Mid    float64
	Micro  float64
	Spread float64
	Tick   float64

	BestBid float64
	BestAsk float64

	ImbalanceTop3 float64
	ImbalanceTop5 float64

	QDBid float64
	QDAsk float64
	QRBid float64
	QRAsk float64

	BookOFINorm    float64
	TradeOFINorm   float64
	NetOFINorm     float64
	MicroDriftNorm float64

	Return1s   float64
	Vol10s     float64
	Entropy10s float64

	IsingField        float64
	IsingMagnet       float64
	IsingEnergy       float64
	IsingProbUp       float64
	IsingProbDown     float64
	IsingSuscept      float64
	IsingSpin         int
	IsingConsensus    float64
	IsingCriticalness float64

	UpdatedAt time.Time
}

type RegimeState struct {
	Tradable bool
	Reason   string
}

type Signal struct {
	Symbol string

	LongScore  float64
	ShortScore float64

	Long    bool
	Short   bool
	NoTrade bool
	Reason  string

	ProbLong  float64
	ProbShort float64

	CreatedAt time.Time
}

type Position struct {
	Side      Side
	Entry     float64
	EntryTime time.Time
	Size      float64

	EntryBid float64
	EntryAsk float64
}
