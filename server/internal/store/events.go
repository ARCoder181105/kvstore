package store

import "time"

type EventType string

type Event struct {
	Type      EventType `json:"type"`
	Key       string    `json:"key"`
	Value     string    `json:"value"`
	TTL       int64     `json:"ttl"`
	Timestamp time.Time `json:"timestamp"`
}

const (
	EventSet     EventType = "SET"
	EventDel     EventType = "DEL"
	EventExpire  EventType = "EXPIRE"
	EventExpired EventType = "EXPIRED"
)
