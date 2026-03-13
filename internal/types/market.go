package types

import "time"

type Side int

const (
	SideUnknown Side = iota
	SideBuy
	SideSell
)

func (s Side) String() string {
	switch s {
	case SideBuy:
		return "buy"
	case SideSell:
		return "sell"
	default:
		return "unknown"
	}
}

type Level struct {
	Price float64
	Qty   float64
}

type BookSnapshot struct {
	Symbol    string
	Bids      []Level // desc
	Asks      []Level // asc
	Timestamp time.Time
}

type TradeTick struct {
	Symbol    string
	Price     float64
	Qty       float64
	Side      Side
	Timestamp time.Time
}
