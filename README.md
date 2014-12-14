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
	"github.com/liamcurry/websocket"
	"honnef.co/go/js/console"
)

func main() {
	master := websocket.New("http://localhost:9001/ws")
	//master := websocket.NewWithGlobal("SockJS", "http://localhost:9001/ws")

	master.OnOpen(func(obj js.Object) {
		console.Log("opened", obj)
	})

	master.OnError(func(obj js.Object) {
		console.Log("error", obj)
	})

	master.OnClose(func(obj js.Object) {
		console.Log("closed", obj)
	})

	master.OnMessage(func(e *dom.MessageEvent) {
		console.Log("message", e)
	})
}
```
