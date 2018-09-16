package main

import (
	"crypto/rand"
	"io"
	"net/http"
	"time"

	"golang.org/x/net/websocket"
)

func main() {
	// Serve test folder.
	http.Handle("/", http.FileServer(http.Dir("./test/")))

	http.Handle("/ws/immediate-close", websocket.Handler(func(ws *websocket.Conn) {
		// Cleanly close the connection.
		err := ws.Close()
		if err != nil {
			panic(err)
		}
	}))

	http.Handle("/ws/binary-static", websocket.Handler(func(ws *websocket.Conn) {
		err := websocket.Message.Send(ws, []byte{0x00, 0x01, 0x02, 0x03, 0x04})
		if err != nil {
			panic(err)
		}
	}))

	http.Handle("/ws/multiframe-static", websocket.Handler(func(ws *websocket.Conn) {
		err := websocket.Message.Send(ws, []byte{0x00, 0x01, 0x02})
		if err != nil {
			panic(err)
		}
		time.Sleep(500 * time.Millisecond)
		err = websocket.Message.Send(ws, []byte{0x03, 0x04})
		if err != nil {
			panic(err)
		}
	}))

	http.Handle("/ws/random-1mb", websocket.Handler(func(ws *websocket.Conn) {
		for i := 0; i < 4; i++ {
			data := make([]byte, 256*1024)
			n, err := io.ReadAtLeast(rand.Reader, data, len(data))
			if err != nil {
				panic(err)
			}

			data = data[:n]

			err = websocket.Message.Send(ws, data)
			if err != nil {
				panic(err)
			}
		}
	}))

	http.Handle("/ws/wait-30s", websocket.Handler(func(ws *websocket.Conn) {
		<-time.After(30 * time.Second)
	}))

	err := http.ListenAndServe(":3000", nil)
	if err != nil {
		panic(err)
	}
}
