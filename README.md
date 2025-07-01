# Simple TCP Chat Server in Go Introduction

This is a basic TCP chat server implemented in Go without 3rd party libraries. It supports multiple clients connecting concurrently, broadcasting messages between clients, and has upload/download byte limits per client. The idea was to make it work using only golang CSP model, without Mutex or Wait groups.

---

## Features

- Concurrent TCP client handling
- Client registration and unregistration
- Message broadcasting to all connected clients except the sender
- Upload/download byte limits per client (configurable)
- Graceful connection handling and logging

---

## Setup

Golang should be installed!

To start server directly use:

```shell script
make run
```

To build binary use:

```shell script
make build
```

To run all tests use:

```shell script
make test
```

## Server

Once the server is up and running, connection are accepted, easiest way to connect is using netcat:

```shell script
nc localhost 9000
```
