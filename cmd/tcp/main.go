package main

import (
	"log"
	"net"

	"github.com/Arun445/tcp-go/internal/config"
	"github.com/Arun445/tcp-go/internal/room"
)

func main() {
	serverConfig := config.Server()
	roomConfig := config.Room()

	listener, err := net.Listen("tcp", serverConfig.Port)
	if err != nil {
		log.Fatalf("Failed to listen on %s: %v", serverConfig.Port, err)
	}
	defer listener.Close()

	room := room.NewRoom(roomConfig)

	go room.Open()

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Accept error: %v", err)
			continue
		}
		go room.NewSession(conn)
	}
}
