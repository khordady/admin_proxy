package main

import (
	"fmt"
	"github.com/gorilla/websocket"
	"log"
	"net"
	"net/url"
)

var server = "192.168.1.105"
var address = "/websocket"
var token = "1234"
var org = "4567"

func main() {
	go makeConnection("20241")
	go makeConnection("20242")
	go makeConnection("3309")

	select {}
}

func makeConnection(local_port string) {
	for {
		// Establish WebSocket connection to VPS
		vpsURL := url.URL{Scheme: "ws", Host: server + ":9090", Path: address + "/admin"}
		query := vpsURL.Query()
		query.Set("port", local_port)
		vpsURL.RawQuery = query.Encode()

		ws, _, err := websocket.DefaultDialer.Dial(vpsURL.String(), nil)
		if err != nil || ws == nil {
			log.Printf("Failed to connect to VPS: %v", err)
			continue
		}

		fmt.Println("Connected WEB Port:", local_port)

		err = ws.WriteMessage(websocket.BinaryMessage, []byte(`{"Token":"`+token+`","ORG":"`+org+`"}`))
		if err != nil {
			log.Printf("Failed to send handshake: %v", err)
			ws.Close()
			continue
		}

		fmt.Println("Sent WEB Port:", local_port)

		//read RND and ORG from server and send it to admin
		_, message, err := ws.ReadMessage()
		if err != nil {
			log.Printf("WebSocket connection closed: %v", err)
			ws.Close()
			continue
		}

		fmt.Println("Read WEB Port:", local_port)

		tcpConn, err := net.Dial("tcp", "localhost:"+local_port)
		if err != nil {
			log.Printf("Failed to connect to message local_port: %v", err)
		}

		fmt.Println("Connected TCP Port:", local_port)

		if local_port != "3309" {
			_, err = tcpConn.Write(message)
			if err != nil {
				log.Printf("Failed to write data to TCP: %v", err)
				ws.Close()
				tcpConn.Close()
				continue
			}
		}

		fmt.Println(message)

		go handleConnection(tcpConn, ws)
	}
}

func handleConnection(tcpConn net.Conn, ws *websocket.Conn) {
	defer tcpConn.Close()
	defer ws.Close()

	// Goroutine to forward data from TCP to WebSocket
	go func() {
		defer tcpConn.Close()
		defer ws.Close()

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
