package store

import "time"

type EventType string

type Event struct {
	Type      EventType
	Key       string
	Value     string
	TTL       int64
	Timestamp time.Time
}

const (
	EventSet     EventType = "SET"
	EventDel     EventType = "DEL"
	EventExpire  EventType = "EXPIRE"
	EventExpired EventType = "EXPIRED"
)
