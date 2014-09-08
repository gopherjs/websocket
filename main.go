package websocket

import (
	"errors"

	"honnef.co/go/js/dom"

	"github.com/gopherjs/gopherjs/js"
)

type readyState int

const (
	connecting readyState = iota // The connection is not yet open.
	open                         // The connection is open and ready to communicate.
	closing                      // The connection is in the process of closing.
	closed                       // The connection is closed or couldn't be opened.
)

type WebSocket struct {
	js.Object
}

func New(url string) *WebSocket {
	object := js.Global.Get("WebSocket").New(url)
	ws := &WebSocket{Object: object}
	return ws
}

func (ws *WebSocket) OnOpen(listener func(js.Object)) {
	ws.Object.Set("onopen", listener)
}

func (ws *WebSocket) OnClose(listener func(js.Object)) {
	ws.Object.Set("onclose", listener)
}

func (ws *WebSocket) OnMessage(listener func(messageEvent *dom.MessageEvent)) {
	wrapper := func(object js.Object) { listener(&dom.MessageEvent{BasicEvent: &dom.BasicEvent{Object: object}}) }
	ws.Object.Set("onmessage", wrapper)
}

func (ws *WebSocket) Send(data string) (err error) {
	defer func() {
		e := recover()
		if e == nil {
			return
		}
		if jsErr, ok := e.(*js.Error); ok && jsErr != nil {
			println(jsErr.Object.Get("name").Str() == "InvalidStateError")
			err = errors.New("InvalidStateError")
		} else {
			panic(e)
		}
	}()
	ws.Object.Call("send", data)
	return nil
}

func (ws *WebSocket) Close() (err error) {
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
