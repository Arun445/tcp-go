package main

import (
	"fmt"
	"net"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Arun445/tcp-go/internal/config"
	"github.com/Arun445/tcp-go/internal/room"
)

// Test helper function to create a testable server
func createTestServer() (*net.Listener, *room.Room, error) {
	serverConfig := &config.ServerConfig{Port: ":9000"}
	roomConfig := &config.RoomConfig{ByteLimit: 100}

	listener, err := net.Listen("tcp", serverConfig.Port)
	if err != nil {
		return nil, nil, err
	}

	room := room.NewRoom(roomConfig)
	go room.Open()

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			go room.NewSession(conn)
		}
	}()

	return &listener, room, nil
}

func TestServer_Integration(t *testing.T) {
	listener, _, err := createTestServer()
	if err != nil {
		t.Fatalf("Failed to create test server")
	}
	defer (*listener).Close()

	addr := (*listener).Addr().(*net.TCPAddr)
	port := addr.Port

	client1, err := net.Dial("tcp", fmt.Sprintf("localhost:%d", port))
	if err != nil {
		t.Fatalf("Failed to connect client1")
	}
	defer client1.Close()

	client2, err := net.Dial("tcp", fmt.Sprintf("localhost:%d", port))
	if err != nil {
		t.Fatalf("Failed to connect client2")
	}
	defer client2.Close()

	time.Sleep(10 * time.Millisecond)

	message := "From client1!"
	_, err = client1.Write([]byte(message))
	if err != nil {
		t.Fatalf("Failed to send message")
	}

	buffer := make([]byte, 1024)
	client2.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
	n, err := client2.Read(buffer)
	if err != nil {
		t.Fatalf("Failed to read message")
	}

	received := string(buffer[:n])
	if received != message {
		t.Errorf("Wrong message recieved")
	}
}

func TestServer_MultipleClients_Concurrent(t *testing.T) {
	listener, _, err := createTestServer()
	if err != nil {
		t.Fatalf("Failed to create test server")
	}
	defer (*listener).Close()

	addr := (*listener).Addr().(*net.TCPAddr)
	port := addr.Port

	numClients := 5
	clients := make([]net.Conn, numClients)

	for i := 0; i < numClients; i++ {
		client, err := net.Dial("tcp", fmt.Sprintf("localhost:%d", port))
		if err != nil {
			t.Fatalf("Failed to connect client")
		}
		clients[i] = client
		defer client.Close()
	}

	time.Sleep(20 * time.Millisecond)

	var wg sync.WaitGroup
	messages := make(chan string, numClients*numClients)
	errors := make(chan error, numClients)

	for i := 1; i < numClients; i++ {
		wg.Add(1)
		go func(clientIndex int) {
			defer wg.Done()
			buffer := make([]byte, 1024)
			clients[clientIndex].SetReadDeadline(time.Now().Add(200 * time.Millisecond))
			n, err := clients[clientIndex].Read(buffer)
			if err != nil {
				errors <- fmt.Errorf("client %d read error:", clientIndex)
				return
			}
			messages <- string(buffer[:n])
		}(i)
	}

	testMessage := "Broadcast test message"
	_, err = clients[0].Write([]byte(testMessage))
	if err != nil {
		t.Fatalf("Failed to send message")
	}

	wg.Wait()
	close(messages)
	close(errors)

	for err := range errors {
		t.Error(err)
	}

	receivedCount := 0
	for msg := range messages {
		if msg != testMessage {
			t.Errorf("Expected '%s', got '%s'", testMessage, msg)
		}
		receivedCount++
	}

	expectedCount := numClients - 1 // All clients except sender
	if receivedCount != expectedCount {
		t.Errorf("Expected %d messages, got %d", expectedCount, receivedCount)
	}
}

func TestServer_UploadLimit_Integration(t *testing.T) {
	listener, _, err := createTestServer()
	if err != nil {
		t.Fatalf("Failed to create test server")
	}
	defer (*listener).Close()

	addr := (*listener).Addr().(*net.TCPAddr)
	port := addr.Port

	client, err := net.Dial("tcp", fmt.Sprintf("localhost:%d", port))
	if err != nil {
		t.Fatalf("Failed to connect client")
	}
	defer client.Close()

	time.Sleep(10 * time.Millisecond)

	largeMessage := make([]byte, 100)
	for i := range largeMessage {
		largeMessage[i] = 'A'
	}

	_, err = client.Write(largeMessage)
	if err != nil {
		t.Fatalf("Failed to send large message")
	}

	buffer := make([]byte, 1024)
	client.SetReadDeadline(time.Now().Add(200 * time.Millisecond))
	n, err := client.Read(buffer)
	if err != nil {
		t.Fatalf("Failed to read limit message")
	}

	received := string(buffer[:n])
	if !strings.Contains(received, "Upload limit reached") {
		t.Errorf("Expected upload limit message")
	}
}
