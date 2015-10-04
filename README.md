websocket
=========

Package websocket provides high- and low-level bindings for the browser's WebSocket API.

The high-level bindings act like a regular net.Conn. They can be used as such. For example:

```Go
c, err := websocket.Dial("ws://localhost/socket") // Blocks until connection is established
if err != nil { handleError() }

buf := make([]byte, 1024)
n, err = c.Read(buf) // Blocks until a WebSocket frame is received
if err != nil { handleError() }
doSomethingWithData(buf[:n])

_, err = c.Write([]byte("Hello!"))
if err != nil { handleError() }

err = c.Close()
if err != nil { handleError() }
```

The low-level bindings use the typical JavaScript idioms.

```Go
ws, err := websocket.New("ws://localhost/socket") // Does not block.
if err != nil { handleError() }

onOpen := func(ev *js.Object) {
    err := ws.Send([]byte("Hello!")) // Send as a binary frame
    err := ws.Send("Hello!") // Send a text frame
}

ws.AddEventListener("open", false, onOpen)
ws.AddEventListener("message", false, onMessage)
ws.AddEventListener("close", false, onClose)
ws.AddEventListener("error", false, onError)

err = ws.Close()
if err != nil { handleError() }
```
