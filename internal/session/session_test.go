package session

import (
	"net"
	"strings"
	"testing"
	"time"

	"github.com/Arun445/tcp-go/internal/message"
)

func TestSession_HandleRead(t *testing.T) {
	serverConn, clientConn := net.Pipe()

	session := &Session{
		ID:   "test-session",
		Conn: serverConn,
	}

	messages := make(chan message.Message, 10)
	done := make(chan struct{})

	go func() {
		session.HandleRead(messages, 1000)
		close(done)
	}()

	testData := []byte("test message")
	_, err := clientConn.Write(testData)
	if err != nil {
		t.Fatalf("Failed to write to client connection")
	}

	select {
	case message := <-messages:
		if message.SessionID != "test-session" {
			t.Errorf("Expected session ID 'test-session', got %s", message.SessionID)
		}
		if string(message.Body) != "test message" {
			t.Errorf("Expected 'test message', got %s", string(message.Body))
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("No message received")
	}

	clientConn.Close()

	select {
	case <-done:
		// Expected - HandleRead should exit on connection close
	case <-time.After(100 * time.Millisecond):
		t.Error("HandleRead did not exit after connection close")
	}
}

func TestSession_HandleRead_Error(t *testing.T) {
	serverConn, clientConn := net.Pipe()

	session := &Session{
		ID:   "test-session",
		Conn: serverConn,
	}

	messages := make(chan message.Message, 10)
	done := make(chan struct{})

	go func() {
		session.HandleRead(messages, 1000)
		close(done)
	}()

	clientConn.Close()

	select {
	case <-done:
		// Expected - HandleRead should return on read error
	case <-time.After(100 * time.Millisecond):
		t.Fatal("HandleRead did not return after connection close")
	}

	select {
	case <-messages:
		t.Errorf("Unexpected message received after connection close")
	case <-time.After(10 * time.Millisecond):
		// Expected - no messages after connection close
	}
}

func TestSession_HandleRead_Multiple(t *testing.T) {
	serverConn, clientConn := net.Pipe()
	defer serverConn.Close()
	defer clientConn.Close()

	session := &Session{
		ID:   "test-session",
		Conn: serverConn,
	}

	messages := make(chan message.Message, 10)
	done := make(chan struct{})

	go func() {
		session.HandleRead(messages, 1000)
		close(done)
	}()

	testData1 := []byte("message1")
	testData2 := []byte("message2")

	_, err := clientConn.Write(testData1)
	if err != nil {
		t.Fatalf("Failed to write first message")
	}

	_, err = clientConn.Write(testData2)
	if err != nil {
		t.Fatalf("Failed to write second message")
	}

	select {
	case message := <-messages:
		if string(message.Body) != "message1" {
			t.Errorf("Expected 'message1', got %s", string(message.Body))
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("No first message received")
	}

	select {
	case message := <-messages:
		if string(message.Body) != "message2" {
			t.Errorf("Expected 'message2', got %s", string(message.Body))
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("No second message received")
	}
}

func TestSession_HandleRead_UploadLimit(t *testing.T) {
	byteLimit := 10
	serverConn, clientConn := net.Pipe()
	defer clientConn.Close()

	session := &Session{
		ID:   "test-session",
		Conn: serverConn,
	}

	messages := make(chan message.Message, 10)
	done := make(chan struct{})

	go func() {
		session.HandleRead(messages, byteLimit)
		close(done)
	}()

	testData := []byte("This message exceeds the upload limit")
	_, err := clientConn.Write(testData)
	if err != nil {
		t.Fatalf("Failed to write to client connection: %v", err)
	}

	buffer := make([]byte, 200)
	clientConn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
	bytesRead, err := clientConn.Read(buffer)
	if err != nil {
		t.Fatalf("Failed to read from client connection")
	}

	received := string(buffer[:bytesRead])
	if !strings.Contains(received, "Upload limit reached") {
		t.Errorf("Expected upload limit message, got: %s", received)
	}

	select {
	case <-done:
		// Expected - HandleRead should exit after limit
	case <-time.After(100 * time.Millisecond):
		t.Error("HandleRead did not exit after upload limit reached")
	}
}

func TestSession_HandleRead_UploadLimitExactBoundary(t *testing.T) {
	byteLimit := 10
	serverConn, clientConn := net.Pipe()
	defer clientConn.Close()

	session := &Session{
		ID:            "test-session",
		Conn:          serverConn,
		UploadedBytes: 5,
	}

	messages := make(chan message.Message, 10)
	done := make(chan struct{})

	go func() {
		session.HandleRead(messages, byteLimit)
		close(done)
	}()

	testData := []byte("5byte")
	_, err := clientConn.Write(testData)
	if err != nil {
		t.Fatalf("Failed to write to client connection")
	}

	// Read the upload limit message
	buffer := make([]byte, 200)
	clientConn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
	bytesRead, err := clientConn.Read(buffer)
	if err != nil {
		t.Fatalf("Failed to read from client connection")
	}

	received := string(buffer[:bytesRead])
	if !strings.Contains(received, "Upload limit reached") {
		t.Errorf("Expected upload limit message, got: %s", received)
	}

	select {
	case <-done:
		// Expected - HandleRead should exit after limit
	case <-time.After(100 * time.Millisecond):
		t.Error("HandleRead did not exit after upload limit reached")
	}
}

func TestSession_HandleWrite(t *testing.T) {
	serverConn, clientConn := net.Pipe()
	defer clientConn.Close()

	session := &Session{
		ID:       "test-session",
		Conn:     serverConn,
		Messages: make(chan []byte, 10),
		Done:     make(chan struct{}),
	}

	testMessage := []byte("Im alive!")
	session.Messages <- testMessage

	done := make(chan struct{})
	go func() {
		session.HandleWrite(1000)
		close(done)
	}()

	buffer := make([]byte, 100)
	clientConn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
	bytesRead, err := clientConn.Read(buffer)
	if err != nil {
		t.Fatalf("Failed to read from client connection")
	}

	received := string(buffer[:bytesRead])
	if received != "Im alive!" {
		t.Errorf("Expected 'Im alive!', got %s", received)
	}

	close(session.Done)

	select {
	case <-done:
		// Expected - HandleWrite should exit when Done is closed
	case <-time.After(100 * time.Millisecond):
		t.Error("HandleWrite did not exit after Done channel closed")
	}
}

func TestSession_HandleWrite_Error(t *testing.T) {
	serverConn, clientConn := net.Pipe()

	session := &Session{
		ID:       "test-session",
		Conn:     serverConn,
		Messages: make(chan []byte, 10),
		Done:     make(chan struct{}),
	}

	clientConn.Close()

	testMessage := []byte("test")
	session.Messages <- testMessage

	done := make(chan struct{})
	go func() {
		session.HandleWrite(1000)
		close(done)
	}()

	select {
	case <-done:
		// Expected - HandleWrite should return due to write error
	case <-time.After(100 * time.Millisecond):
		t.Fatal("HandleWrite did not return after write error")
	}
}

func TestSession_HandleWrite_DownloadLimit(t *testing.T) {
	serverConn, clientConn := net.Pipe()
	defer clientConn.Close()

	session := &Session{
		ID:       "test-session",
		Conn:     serverConn,
		Messages: make(chan []byte, 10),
		Done:     make(chan struct{}),
	}

	testMessage := []byte("This message is longer than 10 bytes")
	session.Messages <- testMessage

	limit := 10
	done := make(chan struct{})
	go func() {
		session.HandleWrite(limit)
		close(done)
	}()

	buffer := make([]byte, 200)
	clientConn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
	bytesRead, err := clientConn.Read(buffer)
	if err != nil {
		t.Fatalf("Failed to read from client connection")
	}

	received := string(buffer[:bytesRead])
	if !strings.Contains(received, "Download limit reached") {
		t.Errorf("Expected download limit message, got: %s", received)
	}

	select {
	case <-done:
		// Expected - HandleWrite should exit after limit
	case <-time.After(100 * time.Millisecond):
		t.Error("HandleWrite did not exit after download limit reached")
	}
}

func TestSession_HandleWrite_DownloadLimitExactBoundary(t *testing.T) {
	byteLimit := 10
	serverConn, clientConn := net.Pipe()
	defer clientConn.Close()

	session := &Session{
		ID:              "test-session",
		Conn:            serverConn,
		Messages:        make(chan []byte, 10),
		Done:            make(chan struct{}),
		DownloadedBytes: 5,
	}

	testMessage := []byte("5byte")
	session.Messages <- testMessage

	done := make(chan struct{})
	go func() {
		session.HandleWrite(byteLimit)
		close(done)
	}()

	buffer := make([]byte, 200)
	clientConn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
	bytesRead, err := clientConn.Read(buffer)
	if err != nil {
		t.Fatalf("Failed to read from client connection")
	}

	received := string(buffer[:bytesRead])
	if !strings.Contains(received, "Download limit reached") {
		t.Errorf("Expected download limit message")
	}

	select {
	case <-done:
		// Expected - HandleWrite should exit after limit
	case <-time.After(100 * time.Millisecond):
		t.Error("HandleWrite did not exit after download limit reached")
	}
}

func TestSession_HandleWrite_ChannelClosed(t *testing.T) {
	serverConn, clientConn := net.Pipe()
	defer clientConn.Close()

	session := &Session{
		ID:       "test-session",
		Conn:     serverConn,
		Messages: make(chan []byte, 10),
		Done:     make(chan struct{}),
	}

	close(session.Messages)

	done := make(chan struct{})
	go func() {
		session.HandleWrite(1000)
		close(done)
	}()

	select {
	case <-done:
		// Expected - HandleWrite should exit when Messages channel is closed
	case <-time.After(100 * time.Millisecond):
		t.Error("HandleWrite did not exit after Messages channel closed")
	}
}

func TestSession_HandleWrite_DoneChannelClosed(t *testing.T) {
	serverConn, clientConn := net.Pipe()
	defer clientConn.Close()

	session := &Session{
		ID:       "test-session",
		Conn:     serverConn,
		Messages: make(chan []byte, 10),
		Done:     make(chan struct{}),
	}

	close(session.Done)

	done := make(chan struct{})
	go func() {
		session.HandleWrite(1000)
		close(done)
	}()

	select {
	case <-done:
		// Expected - HandleWrite should exit immediately
	case <-time.After(100 * time.Millisecond):
		t.Error("HandleWrite did not exit after Done channel was closed")
	}
}
