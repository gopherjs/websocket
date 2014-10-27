package main

import (
	"fmt"

	"github.com/nightexcessive/websocket"
)

func testMessage(socket *websocket.WebSocket, message string) {
	if err := socket.SendString("Hello, World!"); err != nil {
		panic(fmt.Sprintf("Error when sending: %s\n", err))
		return
	}
	fmt.Printf("Message sent: %q\n", message)

	messageEvent, errorEvent := socket.Receive()
	if errorEvent != nil {
		panic(fmt.Sprintf("Error when receiving: %s\n", errorEvent))
		return
	} else if receivedMessage := messageEvent.Data.Str(); receivedMessage != message {
		fmt.Printf("Received unexecpected message: %q (expected %q)\n", receivedMessage, message)
		return
	}
	fmt.Printf("Message received: %#v\n", messageEvent.Data.Interface())
}

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

		testMessage(socket, "Hello, World!")
	}()
}
