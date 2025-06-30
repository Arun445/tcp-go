package room

import (
	"fmt"
	"log"
	"net"
	"time"

	"github.com/Arun445/tcp-go/internal/config"
	"github.com/Arun445/tcp-go/internal/message"
	"github.com/Arun445/tcp-go/internal/session"
)

func NewRoom(roomConfig *config.RoomConfig) *Room {
	return &Room{
		config:   roomConfig,
		events:   make(chan Event),
		messages: make(chan message.Message),
		sessions: make(map[string]*session.Session),
	}
}

func (room *Room) NewSession(conn net.Conn) {
	// uuid preferred
	sessionID := fmt.Sprintf("%s-%d", conn.RemoteAddr(), time.Now().Unix())
	session := &session.Session{
		ID:       sessionID,
		Conn:     conn,
		Messages: make(chan []byte),
		Done:     make(chan struct{}),
	}

	room.events <- Event{Session: session, Type: Register}

	go session.HandleWrite(room.config.ByteLimit)
	session.HandleRead(room.messages, room.config.ByteLimit)

	close(session.Done)

	room.events <- Event{Session: session, Type: Unregister}
}

func (room *Room) Open() {
	for {
		select {
		case event := <-room.events:
			if event.Type == Register {
				room.sessions[event.Session.ID] = event.Session
				log.Printf("Session registered: %s", event.Session.ID)
			}
			if event.Type == Unregister {
				if _, ok := room.sessions[event.Session.ID]; ok {
					delete(room.sessions, event.Session.ID)
					close(event.Session.Messages)
					log.Printf("Session unregistered: %s", event.Session.ID)
				}
			}

		case m := <-room.messages:
			for _, session := range room.sessions {
				if m.SessionID != session.ID {
					session.Messages <- m.Body
				}
			}
		}
	}
}
