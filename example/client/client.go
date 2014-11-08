package main

import (
	"bytes"
	"fmt"

	"github.com/gopherjs/gopherjs/js"
	"honnef.co/go/js/console"
	"honnef.co/go/js/dom"

	"github.com/gopherjs/websocket"
)

func testEcho(socket *websocket.WebSocket, message []byte) {
	console.Time(fmt.Sprintf("WebSocket: Echo (binary): % x", message))
	if _, err := socket.Write(message); err != nil {
		panic(fmt.Sprintf("Error when sending: %s\n", err))
		return
	}

	incomingMessage := make([]byte, len(message))
	n, errorEvent := socket.Read(incomingMessage)
	if errorEvent != nil {
		panic(fmt.Sprintf("Error when receiving: %s\n", errorEvent))
		return
	}
	receivedBytes := incomingMessage[:n]

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

		/*defer func() {
			socket.Close()
			console.Log("WebSocket: Disconnected")
		}()*/

		testEcho(socket, []byte("Hello, World!"))
		testEcho(socket, []byte("World, Hello!"))
		testEcho(socket, []byte{0x01, 0x02, 0x03})
		js.Global.Set("testSocket", socket)
	}()
}
