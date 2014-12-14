websocket
=========

Package websocket will provide GopherJS bindings for the WebSocket API.

It is currently in development and should not be considered stable. The public API is unfinished and will change.

Example
-------

```go
package main

import (
	"github.com/gopherjs/gopherjs/js"
	"github.com/gopherjs/websocket"
	"honnef.co/go/js/console"
)

func main() {
	ws := websocket.New("http://localhost:9001/ws")
	//ws := websocket.NewWithGlobal("SockJS", "http://localhost:9001/ws")

	ws.OnOpen(func(obj js.Object) {
		console.Log("opened", obj)
	})

	ws.OnError(func(obj js.Object) {
		console.Log("error", obj)
	})

	ws.OnClose(func(obj js.Object) {
		console.Log("closed", obj)
	})

	ws.OnMessage(func(obj js.Object) {
		console.Log("message", obj)
	})
}
```
