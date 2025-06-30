package room

import "github.com/Arun445/tcp-go/internal/session"

type SessionEventType int

const (
	Register SessionEventType = iota
	Unregister
)

type Event struct {
	Session *session.Session
	Type    SessionEventType
}
