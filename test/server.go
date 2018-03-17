package main

import (
	"net/http"
	"strings"
	"time"

	"golang.org/x/net/websocket"
)

func negotiateProtocol(offered []string, supported []string) []string {
	for _, supported := range supported {
		for _, offered := range offered {
			if offered == supported {
				return []string{offered}
			}
		}
	}
	return offered
}

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

	http.Handle("/ws/subprotocols", &websocket.Server{
		Handshake: func(c *websocket.Config, r *http.Request) error {
			c.Protocol = negotiateProtocol(c.Protocol, []string{"mqtt", "mqttv3.1"})
			return nil
		},
		Handler: websocket.Handler(func(ws *websocket.Conn) {
			err := websocket.Message.Send(ws, []byte(strings.Join(ws.Config().Protocol, ",")))
			if err != nil {
				panic(err)
			}
		}),
	})

	http.Handle("/ws/wait-30s", websocket.Handler(func(ws *websocket.Conn) {
		<-time.After(30 * time.Second)
	}))

	err := http.ListenAndServe(":3000", nil)
	if err != nil {
		panic(err)
	}
}
