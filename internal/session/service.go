package session

import (
	"log"

	"github.com/Arun445/tcp-go/internal/message"
)

func (session *Session) HandleWrite(limit int) {
	defer session.Conn.Close()

	for {
		select {
		case <-session.Done:
			return
		case message, ok := <-session.Messages:
			if !ok {
				return
			}

			session.DownloadedBytes += len(message)
			if session.DownloadedBytes >= limit {
				session.Conn.Write([]byte("Download limit reached. Disconnecting...\n"))
				return
			}
			if _, err := session.Conn.Write(message); err != nil {
				log.Printf("Error writing to session %s: %v", session.ID, err)
				return
			}
		}
	}
}

func (session *Session) HandleRead(messages chan<- message.Message, limit int) {
	buffer := make([]byte, 1024)

	for {
		bytesRead, err := session.Conn.Read(buffer)
		if err != nil {
			log.Printf("Session %s read error: %v", session.ID, err)
			return
		}
		if bytesRead == 0 {
			continue
		}

		session.UploadedBytes += bytesRead
		if session.UploadedBytes >= limit {
			session.Conn.Write([]byte("Upload limit reached. Disconnecting...\n"))
			return
		}

		messages <- message.Message{SessionID: session.ID, Body: buffer[:bytesRead]}
	}
}
