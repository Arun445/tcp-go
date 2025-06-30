package room

import (
	"github.com/Arun445/tcp-go/internal/config"
	"github.com/Arun445/tcp-go/internal/message"
	"github.com/Arun445/tcp-go/internal/session"
)

type Room struct {
	config   *config.RoomConfig
	events   chan Event
	messages chan message.Message
	sessions map[string]*session.Session
}
