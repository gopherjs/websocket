package main

import (
	"io"
	"net/http"
	"os"

	"code.google.com/p/go.net/websocket"
)

func echoHandler(ws *websocket.Conn) {
	io.Copy(ws, ws)
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
