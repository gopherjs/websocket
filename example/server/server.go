package main

import (
	"bytes"
	"fmt"
	"net/http"
	"os"

	"code.google.com/p/go.net/websocket"
)

func echoHandler(ws *websocket.Conn) {
	var strMessage string
	fmt.Println("Receiving first message...")
	websocket.Message.Receive(ws, &strMessage)
	fmt.Printf("Got first message: %q\n", strMessage)
	if strMessage != "Hello, String!" {
		fmt.Println("First message was bad.")
		return
	}
	fmt.Println("Returning first message...")
	websocket.Message.Send(ws, strMessage)

	var byteMessage []byte
	fmt.Println("Receiving second message...")
	websocket.Message.Receive(ws, &byteMessage)
	fmt.Printf("Got second message: % x\n", byteMessage)
	if !bytes.Equal(byteMessage, []byte{0x00, 0x01, 0x02, 0x03, 0x04, 0x05}) {
		fmt.Println("Second message was bad.")
		return
	}
	fmt.Println("Returning second message...")
	websocket.Message.Send(ws, byteMessage)
	fmt.Println("Done.")
}

func main() {
	http.Handle("/echo", websocket.Handler(echoHandler))
	http.Handle("/gopath/", http.StripPrefix("/gopath", http.FileServer(http.Dir(os.Getenv("GOPATH")))))
	http.Handle("/goroot/", http.StripPrefix("/goroot", http.FileServer(http.Dir(os.Getenv("GOROOT")))))
	http.Handle("/", http.FileServer(http.Dir("../client")))
	err := http.ListenAndServe("127.0.0.1:3000", nil)
	if err != nil {
		panic("ListenAndServe: " + err.Error())
	}
}
