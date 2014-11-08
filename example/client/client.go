package main

import (
	"bytes"
	"fmt"

	"honnef.co/go/js/console"
	"honnef.co/go/js/dom"

	"github.com/gopherjs/websocket"
)

func testMessage(socket *websocket.WebSocket, message string) {
	console.Time(fmt.Sprintf("WebSocket: Echo: %#v", message))
	if err := socket.SendString(message); err != nil {
		panic(fmt.Sprintf("Error when sending: %s\n", err))
		return
	}

	messageEvent, errorEvent := socket.Receive()
	if errorEvent != nil {
		panic(fmt.Sprintf("Error when receiving: %s\n", errorEvent))
		return
	}
	if receivedMessage := messageEvent.Data.Str(); receivedMessage != message {
		console.TimeEnd(fmt.Sprintf("WebSocket: Echo: %#v", message))
		console.Warn(fmt.Sprintf("Received unexecpected message: %q (expected %q)", receivedMessage, message))
		return
	}
	console.TimeEnd(fmt.Sprintf("WebSocket: Echo: %#v", message))
}

func testMessageBinary(socket *websocket.WebSocket, message []byte) {
	console.Time(fmt.Sprintf("WebSocket: Echo (binary): % x", message))
	if _, err := socket.Write(message); err != nil {
		panic(fmt.Sprintf("Error when sending: %s\n", err))
		return
	}

	messageEvent, errorEvent := socket.Receive()
	if errorEvent != nil {
		panic(fmt.Sprintf("Error when receiving: %s\n", errorEvent))
		return
	}

	receivedString := messageEvent.Data.Str()
	receivedBytes := []byte(receivedString)

	if bytes.Compare(receivedBytes, message) != 0 {
		console.TimeEnd(fmt.Sprintf("WebSocket: Echo (binary): % x", message))
		console.Warn(fmt.Sprintf("Received unexecpected message: % x (expected % x)", receivedBytes, message))
		return
	}
	console.TimeEnd(fmt.Sprintf("WebSocket: Echo (binary): % x", message))
}

func main() {
	go func() {
		document := dom.GetWindow().Document().(dom.HTMLDocument)
		location := document.Location()

		wsProtocol := "ws"
		if location.Protocol == "https:" {
			wsProtocol = "wss"
		}

		console.Time("WebSocket: Connect")
		socket, err := websocket.New(fmt.Sprintf("%s://%s:%s/echo", wsProtocol, location.Hostname, location.Port))
		if err != nil {
			fmt.Printf("Failed to connect: %s\n", err)
			return
		}
		console.TimeEnd("WebSocket: Connect")

		defer func() {
			socket.Close()
			console.Log("WebSocket: Disconnected")
		}()

		testMessage(socket, "Hello, World!")
		testMessage(socket, "World, Hello!")
		testMessageBinary(socket, []byte{0x01, 0x02, 0x03})
	}()
}
