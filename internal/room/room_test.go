package room

import (
	"fmt"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/Arun445/tcp-go/internal/config"
	"github.com/Arun445/tcp-go/internal/message"
	"github.com/Arun445/tcp-go/internal/session"
)

func TestRoom_Open_RegisterUnregister(t *testing.T) {
	config := &config.RoomConfig{
		ByteLimit: 100,
	}

	room := NewRoom(config)

	done := make(chan struct{})
	go func() {
		room.Open()
		close(done)
	}()

	serverConn, clientConn := net.Pipe()
	defer clientConn.Close()

	testSession := &session.Session{
		ID:       "test-session-1",
		Conn:     serverConn,
		Messages: make(chan []byte, 1),
		Done:     make(chan struct{}),
	}

	// Test registration
	room.events <- Event{Session: testSession, Type: Register}

	time.Sleep(50 * time.Millisecond)
	if len(room.sessions) != 1 {
		t.Errorf("Expected 1 session, got %d", len(room.sessions))
	}

	if room.sessions["test-session-1"] != testSession {
		t.Error("Session not registered correctly")
	}

	// Test unregistration
	room.events <- Event{Session: testSession, Type: Unregister}
	time.Sleep(50 * time.Millisecond)
	if len(room.sessions) != 0 {
		t.Errorf("Expected 0 sessions, got %d", len(room.sessions))
	}
}

func TestRoom_Open_MessageBroadcast(t *testing.T) {
	config := &config.RoomConfig{
		ByteLimit: 100,
	}

	room := NewRoom(config)
	go room.Open()

	serverConn1, clientConn1 := net.Pipe()
	defer clientConn1.Close()

	serverConn2, clientConn2 := net.Pipe()
	defer clientConn2.Close()

	session1 := &session.Session{
		ID:       "session-1",
		Conn:     serverConn1,
		Messages: make(chan []byte, 10),
		Done:     make(chan struct{}),
	}

	session2 := &session.Session{
		ID:       "session-2",
		Conn:     serverConn2,
		Messages: make(chan []byte, 10),
		Done:     make(chan struct{}),
	}

	room.events <- Event{Session: session1, Type: Register}
	room.events <- Event{Session: session2, Type: Register}

	time.Sleep(50 * time.Millisecond)

	if len(room.sessions) != 2 {
		t.Fatalf("Expected 2 sessions, got %d", len(room.sessions))
	}

	testMessage := message.Message{
		SessionID: "session-1",
		Body:      []byte("Message from session 1"),
	}

	room.messages <- testMessage

	time.Sleep(50 * time.Millisecond)

	select {
	case msg := <-session2.Messages:
		if string(msg) != "Message from session 1" {
			t.Errorf("Expected 'Message from session 1', got %s", string(msg))
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("Session2 did not receive message")
	}

	select {
	case <-session1.Messages:
		t.Error("Session1 received its own message")
	case <-time.After(10 * time.Millisecond):
		// should not receive its own message
	}
}

func TestRoom_NewSession_Integration(t *testing.T) {
	config := &config.RoomConfig{
		ByteLimit: 100,
	}

	room := NewRoom(config)
	go room.Open()

	serverConn, clientConn := net.Pipe()

	go room.NewSession(serverConn)

	if len(room.sessions) == 1 {
		t.Errorf("Expected 1 sessions, got %d", len(room.sessions))
	}

	clientConn.Write([]byte("test message"))
	time.Sleep(1000 * time.Millisecond)
	clientConn.Close()

	time.Sleep(100 * time.Millisecond)

	if len(room.sessions) != 0 {
		t.Errorf("Expected 0 sessions, got %d", len(room.sessions))
	}
}

func TestRoom_MessageBroadcast_MultipleSessions(t *testing.T) {
	config := &config.RoomConfig{
		ByteLimit: 1000,
	}

	room := NewRoom(config)
	go room.Open()

	numSessions := 5
	sessions := make([]*session.Session, numSessions)
	serverConns := make([]net.Conn, numSessions)
	clientConns := make([]net.Conn, numSessions)

	for i := 0; i < numSessions; i++ {
		serverConn, clientConn := net.Pipe()
		serverConns[i] = serverConn
		clientConns[i] = clientConn

		session := &session.Session{
			ID:       fmt.Sprintf("session-%d", i),
			Conn:     serverConn,
			Messages: make(chan []byte, 10),
			Done:     make(chan struct{}),
		}
		sessions[i] = session
		room.events <- Event{Session: session, Type: Register}
	}

	time.Sleep(100 * time.Millisecond)

	if len(room.sessions) != numSessions {
		t.Fatalf("Expected %d sessions, got %d", numSessions, len(room.sessions))
	}

	testMessage := message.Message{
		SessionID: "session-0",
		Body:      []byte("random message"),
	}
	room.messages <- testMessage

	time.Sleep(100 * time.Millisecond)

	for i := 1; i < numSessions; i++ {
		select {
		case msg := <-sessions[i].Messages:
			if string(msg) != "random message" {
				t.Errorf("Session %d: expected 'random message', got %s", i, string(msg))
			}
		case <-time.After(100 * time.Millisecond):
			t.Errorf("Session %d did not receive message", i)
		}
	}

	select {
	case <-sessions[0].Messages:
		t.Error("Session 0 received its own message")
	case <-time.After(10 * time.Millisecond):
		// expected
	}

	for i := 0; i < numSessions; i++ {
		clientConns[i].Close()
		serverConns[i].Close()
	}
}

func TestRoom_ConcurrentOperations(t *testing.T) {
	config := &config.RoomConfig{
		ByteLimit: 1000,
	}

	room := NewRoom(config)
	go room.Open()

	var wg sync.WaitGroup
	numGoroutines := 10

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			serverConn, clientConn := net.Pipe()
			defer clientConn.Close()
			defer serverConn.Close()

			session := &session.Session{
				ID:       fmt.Sprintf("concurrent-session-%d", id),
				Conn:     serverConn,
				Messages: make(chan []byte, 10),
				Done:     make(chan struct{}),
			}

			room.events <- Event{Session: session, Type: Register}
			time.Sleep(10 * time.Millisecond)

			msg := message.Message{
				SessionID: session.ID,
				Body:      []byte(fmt.Sprintf("message from session %d", id)),
			}
			room.messages <- msg

			time.Sleep(10 * time.Millisecond)

			room.events <- Event{Session: session, Type: Unregister}
		}(i)
	}

	wg.Wait()
	time.Sleep(100 * time.Millisecond)

	if len(room.sessions) != 0 {
		t.Errorf("Expected 0 sessions after cleanup, got %d", len(room.sessions))
	}
}
