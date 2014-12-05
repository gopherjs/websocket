package main

import (
	"go/build"
	"time"

	"code.google.com/p/go.net/websocket"
	"github.com/codegangsta/martini"
)

func main() {
	m := martini.Classic()

	//serve test folder
	m.Use(martini.Static("test"))

	//serve sourcemaps from GOROOT and GOPATH
	m.Use(martini.Static(build.Default.GOROOT, martini.StaticOptions{Prefix: "goroot"}))
	m.Use(martini.Static(build.Default.GOPATH, martini.StaticOptions{Prefix: "gopath"}))

	m.Get("/ws/immediate-close", websocket.Handler(func(ws *websocket.Conn) {
		// Cleanly close the connection.
		if err := ws.Close(); err != nil {
			panic(err)
		}
	}).ServeHTTP)

	m.Get("/ws/binary-static", websocket.Handler(func(ws *websocket.Conn) {
		if err := websocket.Message.Send(ws, []byte{0x00, 0x01, 0x02, 0x03, 0x04}); err != nil {
			panic(err)
		}
	}).ServeHTTP)

	m.Get("/ws/wait-30s", websocket.Handler(func(ws *websocket.Conn) {
		<-time.After(30 * time.Second)
	}).ServeHTTP)

	m.Run()
}
