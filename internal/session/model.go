package session

import "net"

type Session struct {
	ID              string
	Conn            net.Conn
	Messages        chan []byte
	Done            chan struct{}
	DownloadedBytes int
	UploadedBytes   int
}
