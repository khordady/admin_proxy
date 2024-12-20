package main

import (
	"fmt"
	"github.com/gorilla/websocket"
	"log"
	"net"
	"net/url"
)

func main() {
	go makeConnection("20241", "8001")
	go makeConnection("20242", "8002")
	go makeConnection("3309", "8009")

	select {}
}

func makeConnection(local_port, remote_port string) {
	for {
		tcpConn, err := net.Dial("tcp", "localhost:"+local_port)
		if err != nil {
			log.Fatalf("Failed to connect to message local_port: %v", err)
		}

		// Establish WebSocket connection to VPS
		vpsURL := url.URL{Scheme: "ws", Host: "192.168.1.104", Path: "/admin"}
		query := vpsURL.Query()
		query.Set("port", remote_port)
		vpsURL.RawQuery = query.Encode()

		ws, _, err := websocket.DefaultDialer.Dial(vpsURL.String(), nil)
		if err != nil {
			log.Fatalf("Failed to connect to VPS: %v", err)
		}

		//read RND and ORG from server and send it to admin
		_, message, err := ws.ReadMessage()
		if err != nil {
			log.Printf("WebSocket connection closed: %v", err)
			return
		}

		_, err = tcpConn.Write(message)
		if err != nil {
			log.Printf("Failed to write data to TCP: %v", err)
			return
		}

		fmt.Println(message)

		// Start proxying connections
		go handleConnection(tcpConn, ws)
	}
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
