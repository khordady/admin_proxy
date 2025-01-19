package main

import (
	"errors"
	"flag"
	"fmt"
	"github.com/gorilla/websocket"
	"log"
	"net"
	"net/url"
	"os"
	"time"
)

//var server = "192.168.1.105:9090"

var server = "mail.expanel.app:9090"

var address = "/websocket"

//var wss = "ws"

var wss = "wss"

var token *string
var org *string

func main() {
	log.SetOutput(os.Stdout)

	token = flag.String("token", "", "The authentication token")
	org = flag.String("org", "", "The organization name")

	// Parse the flags
	flag.Parse()

	// Check if required arguments are provided
	if *token == "" || *org == "" {
		os.Exit(1)
	}

	go makeConnection("20241")
	go makeConnection("20242")
	makeConnection("3309")
}

func makeConnection(local_port string) {
	for {
		// Establish WebSocket connection to VPS
		vpsURL := url.URL{Scheme: wss, Host: server, Path: address + "/admin"}
		query := vpsURL.Query()
		query.Set("port", local_port)
		vpsURL.RawQuery = query.Encode()

		fmt.Println("Try Connecting WEB : ", vpsURL.String())

		ws, _, err := websocket.DefaultDialer.Dial(vpsURL.String(), nil)
		if err != nil || ws == nil {
			fmt.Printf("Failed to connect to VPS: %v\n", err)
			time.Sleep(3 * time.Second)
			continue
		}

		fmt.Println("Connected WEB Port: ", local_port)

		err = ws.WriteMessage(websocket.BinaryMessage, []byte(`{"Token":"`+*token+`","ORG":"`+*org+`"}`))
		if err != nil {
			fmt.Printf("Failed to send handshake: %v\n", err)
			ws.Close()
			time.Sleep(3 * time.Second)
			continue
		}

		fmt.Println("Sent WEB Port: ", local_port)

		if local_port != "3309" {
			//read start communication to make sure connection is alive
			fmt.Println("Read start message")
			_, message, err3 := ws.ReadMessage()
			if message != nil {
				fmt.Printf("First message is: %s \n", string(message))
			}
			if err3 != nil {
				fmt.Printf("Error read first message %v\n", err3)
				ws.Close()
				time.Sleep(3 * time.Second)
				continue
			}
		}

		//read RND and ORG from server and send it to admin
		n, message, err := ws.ReadMessage()
		if err != nil {
			if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				fmt.Printf("WebSocket closed normally: %v\n", err)
			} else if websocket.IsUnexpectedCloseError(err) {
				fmt.Printf("Unexpected WebSocket error: %v\n", err)
			} else {
				fmt.Printf("Other error: %v\n", err)
			}
			ws.Close()
			time.Sleep(3 * time.Second)
			continue
		}

		fmt.Println("Read WEB Port:", local_port)

		tcpConn, err := net.Dial("tcp", "localhost: "+local_port)
		if err != nil {
			fmt.Printf("Failed to connect to message local_port: %v\n", err)
			ws.Close()
			time.Sleep(3 * time.Second)
			continue
		}

		fmt.Println("Connected TCP Port:", local_port)

		if local_port != "3309" {
			_, err = tcpConn.Write(message)
			if err != nil {
				fmt.Printf("Failed to write data to TCP: %v\n", err)
				ws.Close()
				tcpConn.Close()
				time.Sleep(3 * time.Second)
				continue
			}
		} else {
			fmt.Printf("First message Ports: %s %s size: %d \n", local_port, string(message), n)
		}

		go handleConnection(tcpConn, ws, local_port)
	}
}

func handleConnection(tcpConn net.Conn, ws *websocket.Conn, port string) {
	defer tcpConn.Close()
	defer ws.Close()
	defer log.Printf("Proxy communication finished %s", port)

	// Goroutine to forward data from TCP to WebSocket
	go func() {
		defer tcpConn.Close()
		defer ws.Close()

		buffer := make([]byte, 32*1024)
		for {
			// Read from TCP connection
			n, err := tcpConn.Read(buffer)
			if n > 0 {
				// Forward to WebSocket
				err2 := ws.WriteMessage(websocket.BinaryMessage, buffer[:n])
				if err2 != nil {
					fmt.Printf("Failed to send data to WebSocket: %v\n", err2)
					return
				}
			}
			if err != nil {
				if errors.Is(err, net.ErrClosed) {
					fmt.Println("TCP connection closed by peer")
					return
				} else {
					fmt.Printf("TCP connection read error: %v\n", err)
					return
				}
			}
		}
	}()

	// Forward data from WebSocket to TCP
	for {
		// Read from WebSocket
		_, message, err := ws.ReadMessage()
		if message != nil {
			// Write to TCP connection
			_, err2 := tcpConn.Write(message)
			if err2 != nil {
				fmt.Printf("Failed to write data to TCP: %v\n", err2)
				return
			}
		}
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				fmt.Println("TCP connection closed by peer")
				return
			} else {
				fmt.Printf("WS connection read error: %v\n", err)
				return
			}
		}
	}
}
