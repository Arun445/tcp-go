package config

import (
	"os"
	"strconv"
)

type ServerConfig struct {
	Port string
}

type RoomConfig struct {
	ByteLimit int
}

func Server() *ServerConfig {
	port := os.Getenv("APP_PORT")
	if port == "" {
		port = ":9000"
	}

	return &ServerConfig{
		Port: port,
	}
}

func Room() *RoomConfig {
	byteLimit := 100
	if byteLimitEnv := os.Getenv("BYTE_LIMIT"); byteLimitEnv != "" {
		if byteLimitInt, err := strconv.Atoi(byteLimitEnv); err == nil {
			byteLimit = byteLimitInt
		}
	}
	return &RoomConfig{
		ByteLimit: byteLimit,
	}
}
