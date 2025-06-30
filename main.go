package main

import (
	"fmt"
	"log"
	"net"
	"time"
)

type ClientEventType int

const (
	Register ClientEventType = iota
	Unregister
)

type Client struct {
	id              string
	conn            net.Conn
	messages        chan []byte
	done            chan struct{}
	downloadedBytes int
	uploadedBytes   int
}

type Message struct {
	client *Client
	body   []byte
}

type Event struct {
	client *Client
	event  ClientEventType
}

type App struct {
	events   chan Event
	messages chan Message
	clients  map[string]*Client
}

const (
	port      = ":9000"
	byteLimit = 100
)

func main() {
	tcpListener, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("Failed to start server on port %s: %v", port, err)
	}
	defer tcpListener.Close()

	app := App{
		events:   make(chan Event),
		messages: make(chan Message),
		clients:  make(map[string]*Client),
	}

	go app.listen()

	for {
		conn, err := tcpListener.Accept()
		if err != nil {
			log.Printf("Error accepting connection: %v", err)
			continue
		}

		go app.newClient(conn)
	}
}

func (app *App) newClient(connection net.Conn) {
	// uuid preferred
	clientID := fmt.Sprintf("%s-%d", connection.RemoteAddr().String(), time.Now().Unix())
	newClient := Client{
		id:       clientID,
		conn:     connection,
		messages: make(chan []byte),
		done:     make(chan struct{}),
	}

	app.events <- Event{
		client: &newClient,
		event:  Register,
	}

	go newClient.handleWrite()
	newClient.handleRead(app.messages)

	close(newClient.done)

	app.events <- Event{
		client: &newClient,
		event:  Unregister,
	}
}

func (client *Client) handleWrite() {
	defer client.conn.Close()

	for {
		select {
		case <-client.done:
			return
		case message, ok := <-client.messages:
			if !ok {
				return
			}

			client.downloadedBytes += len(message)

			if client.downloadedBytes >= byteLimit {
				client.conn.Write([]byte("Download limit reached\n"))
				return
			}

			if _, err := client.conn.Write(message); err != nil {
				log.Printf("Error writing to client %s: %v", client.id, err)
				return
			}
		}
	}
}

func (client *Client) handleRead(messages chan<- Message) {
	buffer := make([]byte, 1024)

	for {
		bytesRead, err := client.conn.Read(buffer)
		if err != nil {
			log.Printf("Client %s read error: %v", client.id, err)
			return
		}

		if bytesRead == 0 {
			continue
		}

		client.uploadedBytes += bytesRead

		if client.uploadedBytes >= byteLimit {
			client.conn.Write([]byte("Upload limit reached\n"))
			return
		}

		messages <- Message{client: client, body: buffer[:bytesRead]}
	}
}

func (app *App) listen() {
	for {
		select {
		case event := <-app.events:
			if event.event == Register {
				app.clients[event.client.id] = event.client
				log.Printf("Client registered: %s", event.client.id)
			}

			if event.event == Unregister {
				if _, ok := app.clients[event.client.id]; ok {
					delete(app.clients, event.client.id)
					close(event.client.messages)
					log.Printf("Client unregistered: %s", event.client.id)
				}
			}

		case message := <-app.messages:
			for _, client := range app.clients {
				if message.client.id != client.id {
					client.messages <- message.body
				}
			}
		}
	}
}
