package main

import (
	"fmt"
	"github.com/gorilla/websocket"
	"log"
	"net"
	"net/url"
)

func main() {
	for {
		makeConnection("20241")
		makeConnection("20242")
		makeConnection("3309")
	}
}

func makeConnection(port string) {
	messageConn, err := net.Dial("tcp", "localhost:"+port)
	if err != nil {
		log.Fatalf("Failed to connect to message port: %v", err)
	}

	// Establish WebSocket connection to VPS
	vpsURL := url.URL{Scheme: "wss", Host: "vps.example.com", Path: "/admin"}
	ws, _, err := websocket.DefaultDialer.Dial(vpsURL.String(), nil)
	if err != nil {
		log.Fatalf("Failed to connect to VPS: %v", err)
	}

	// Send KEY for identification
	key := "1234-5678" // Replace with actual key logic
	err = ws.WriteMessage(websocket.TextMessage, []byte(key))
	if err != nil {
		log.Fatalf("Failed to send KEY: %v", err)
	}

	_, message, err := ws.ReadMessage()
	if err != nil {
		log.Printf("WebSocket connection closed: %v", err)
		return
	}

	fmt.Println(message)

	// Start proxying connections
	go handleConnection(messageConn, ws)
}

func handleConnection(tcpConn net.Conn, ws *websocket.Conn) {
	defer tcpConn.Close()
	defer ws.Close()

	// Goroutine to forward data from TCP to WebSocket
	go func() {
		buffer := make([]byte, 32*1024)
		for {
			// Read from TCP connection
			n, err := tcpConn.Read(buffer)
			if err != nil {
				log.Printf("TCP connection closed: %v", err)
				return
			}

			// Forward to WebSocket
			err = ws.WriteMessage(websocket.BinaryMessage, buffer[:n])
			if err != nil {
				log.Printf("Failed to send data to WebSocket: %v", err)
				return
			}
		}
	}()

	// Forward data from WebSocket to TCP
	for {
		// Read from WebSocket
		_, message, err := ws.ReadMessage()
		if err != nil {
			log.Printf("WebSocket connection closed: %v", err)
			return
		}

		// Write to TCP connection
		_, err = tcpConn.Write(message)
		if err != nil {
			log.Printf("Failed to write data to TCP: %v", err)
			return
		}
	}
}
