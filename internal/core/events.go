package core

import "db_trace/kucoin-ising-bot/internal/types"

type EventType int

const (
	EventBook EventType = iota
	EventTrade
)

type Event struct {
	Type  EventType
	Book  *types.BookSnapshot
	Trade *types.TradeTick
}
