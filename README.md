websocket
=========

[![GoDoc](https://godoc.org/github.com/gopherjs/websocket?status.svg)](https://godoc.org/github.com/gopherjs/websocket)

Packages [websocket](https://godoc.org/github.com/gopherjs/websocket) and [websocketjs](https://godoc.org/github.com/gopherjs/websocket/websocketjs) provide high- and low-level bindings for the browser's WebSocket API (respectively).

The high-level bindings offer a Dial function that returns a regular net.Conn.
It can be used similarly to net package.

```Go
conn, err := websocket.Dial("ws://localhost/socket") // Blocks until connection is established.
if err != nil {
	// handle error
}

buf := make([]byte, 1024)
n, err = conn.Read(buf) // Blocks until a WebSocket frame is received.
doSomethingWithData(buf[:n])
if err != nil {
	// handle error
}

_, err = conn.Write([]byte("Hello!"))
// ...

err = conn.Close()
// ...
```

The low-level bindings work with typical JavaScript idioms, such as adding event listeners with callbacks.

```Go
ws, err := websocketjs.New("ws://localhost/socket") // Does not block.
if err != nil {
	// handle error
}

onOpen := func(ev *js.Object) {
	err := ws.Send([]byte("Hello!")) // Send a binary frame.
	// ...
	err := ws.Send("Hello!") // Send a text frame.
	// ...
}

ws.AddEventListener("open", false, onOpen)
ws.AddEventListener("message", false, onMessage)
ws.AddEventListener("close", false, onClose)
ws.AddEventListener("error", false, onError)

err = ws.Close()
// ...
```
