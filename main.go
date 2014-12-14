package websocket

import (
	"errors"

	"github.com/gopherjs/gopherjs/js"
)

type readyState int

const (
	connecting readyState = iota // The connection is not yet open.
	open                         // The connection is open and ready to communicate.
	closing                      // The connection is in the process of closing.
	closed                       // The connection is closed or couldn't be opened.
)

// A WebSocket is anything that implements the websockets interface.
// http://www.w3.org/TR/websockets/#websocket
type WebSocket interface {
	OnOpen(listener func(js.Object))
	OnError(listener func(js.Object))
	OnClose(listener func(js.Object))
	OnMessage(listener func(js.Object))
	Send(data string) error
	Close() error
}

type webSocket struct {
	js.Object
}

// NewWithGlobal allows for customizing the global variable on the window.
func NewWithGlobal(global, url string) WebSocket {
	object := js.Global.Get(global).New(url)
	ws := &webSocket{Object: object}
	return ws
}

// New creates a new websocket.
func New(url string) WebSocket {
	return NewWithGlobal("WebSocket", url)
}

func (ws *webSocket) OnOpen(listener func(js.Object)) {
	ws.Object.Set("onopen", listener)
}

func (ws *webSocket) OnError(listener func(js.Object)) {
	ws.Object.Set("onerror", listener)
}

func (ws *webSocket) OnClose(listener func(js.Object)) {
	ws.Object.Set("onclose", listener)
}

func (ws *webSocket) OnMessage(listener func(js.Object)) {
	wrapper := func(obj js.Object) { listener(obj) }
	ws.Object.Set("onmessage", wrapper)
}

// Thrown when attempting to send data on a websocket that isn't open.
var ErrInvalidState = errors.New("invalid state error")

func (ws *webSocket) Send(data string) (err error) {
	defer func() {
		e := recover()
		if e == nil {
			return
		}
		if jsErr, ok := e.(*js.Error); ok && jsErr != nil {
			println(jsErr.Object.Get("name").Str() == "InvalidStateError")
			err = ErrInvalidState
		} else {
			panic(e)
		}
	}()
	ws.Object.Call("send", data)
	return nil
}

func (ws *webSocket) Close() (err error) {
	defer func() {
		e := recover()
		if e == nil {
			return
		}
		if jsErr, ok := e.(*js.Error); ok && jsErr != nil {
			err = jsErr
		} else {
			panic(e)
		}
	}()
	ws.Object.Call("close")
	return nil
}
