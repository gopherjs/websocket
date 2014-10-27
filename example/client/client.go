package main

import (
	"fmt"

	"github.com/nightexcessive/websocket"
)

func main() {
	go func() {
		fmt.Println("Creating...")
		socket, err := websocket.New("ws://localhost:3000/echo")
		if err != nil {
			fmt.Printf("Failed to connect: %s\n", err)
			return
		}
		defer func() {
			socket.Close()
			fmt.Println("Disconnected.")
		}()
		fmt.Println("Connected.")

		if _, err := socket.Write([]byte{0x01, 0x02, 0x03, 0x04, 0x05}); err != nil {
			fmt.Printf("Error when sending: %s\n", err)
			return
		}

		messageEvent, errorEvent := socket.Receive()
		if errorEvent != nil {
			fmt.Printf("Error when receiving: %s\n", errorEvent)
			return
		}
		fmt.Printf("Message received: %#v\n", messageEvent.Data.Interface())
	}()
}
