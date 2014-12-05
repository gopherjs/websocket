//go:generate gopherjs build -m client.go

package main

import (
	"bytes"
	"fmt"
	"io"

	"honnef.co/go/js/console"
	"honnef.co/go/js/dom"

	"github.com/gopherjs/websocket"
)

func testEchoBytes(c *websocket.Conn, message []byte) {
	fmt.Printf("testEchoBytes: % x\n", message)
	if _, err := c.Write(message); err != nil {
		panic(fmt.Sprintf("Error when sending: %s\n", err))
	}

	incomingMessage := make([]byte, len(message)+1)

	fmt.Println("Read one...")
	n, errorEvent := c.Read(incomingMessage[:len(incomingMessage)-2])
	if errorEvent != nil {
		panic(fmt.Sprintf("Error when receiving: %s\n", errorEvent))
	}

	fmt.Println("Read two...")
	n2, errorEvent := c.Read(incomingMessage[len(incomingMessage)-2:])
	if errorEvent != nil {
		panic(fmt.Sprintf("Error when receiving: %s\n", errorEvent))
	}
	n += n2

	fmt.Printf("Results: % x (%d)\n", incomingMessage, n)

	receivedBytes := incomingMessage[:n]
	fmt.Printf("Finalized: % x\n", receivedBytes)

	if !bytes.Equal(receivedBytes, message) {
		console.Warn(fmt.Sprintf("Received unexecpected message: % x (expected % x)", receivedBytes, message))
		return
	}

	fmt.Println("Done with testEchoBytes.")
}

func testEchoString(c *websocket.Conn, message string) {
	fmt.Printf("testEchoString: %q\n", message)
	if _, err := io.WriteString(c, message); err != nil {
		panic(fmt.Sprintf("Error when sending: %s\n", err))
	}

	incomingMessage := make([]byte, len(message)+1)

	fmt.Println("Read one...")
	n, errorEvent := c.Read(incomingMessage[:len(incomingMessage)-2])
	if errorEvent != nil {
		panic(fmt.Sprintf("Error when receiving: %s\n", errorEvent))
	}

	fmt.Println("Read two...")
	n2, errorEvent := c.Read(incomingMessage[len(incomingMessage)-2:])
	if errorEvent != nil {
		panic(fmt.Sprintf("Error when receiving: %s\n", errorEvent))
	}
	n += n2

	fmt.Printf("Results: % x (%d)\n", incomingMessage, n)

	receivedBytes := incomingMessage[:n]
	fmt.Printf("Finalized: % x\n", receivedBytes)

	receivedString := string(receivedBytes)
	fmt.Printf("Stringified: %s\n", receivedString)

	if receivedString != message {
		console.Warn(fmt.Sprintf("Received unexecpected message: %q (expected %q)", receivedBytes, message))
		return
	}

	fmt.Println("Done with testEchoString.")
}

func stepOne(c *websocket.Conn) {
	testEchoString(c, "Hello, String!")
}

func stepTwo(c *websocket.Conn) {
	testEchoBytes(c, []byte{0x00, 0x01, 0x02, 0x03, 0x04, 0x05})
}

func main() {
	go func() {
		document := dom.GetWindow().Document().(dom.HTMLDocument)
		location := document.Location()

		wsProtocol := "ws"
		if location.Protocol == "https:" {
			wsProtocol = "wss"
		}

		console.Time("Dial")
		c, err := websocket.Dial(fmt.Sprintf("%s://%s:%s/echo", wsProtocol, location.Hostname, location.Port))
		if err != nil {
			fmt.Printf("Failed to connect: %s\n", err)
			return
		}
		console.TimeEnd("Dial")

		defer func() {
			if err := c.Close(); err != nil {
				console.Warn("Error while disconnecting:", err)
				return
			}
			fmt.Println("Cleanly disconnected.")
		}()

		stepOne(c)
		stepTwo(c)
	}()
}
