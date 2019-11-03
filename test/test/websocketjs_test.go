// +build js,!wasm

package websocket_test

import (
	"sync"
	"testing"

	"github.com/LinearZoetrope/testevents"
	"github.com/gopherjs/gopherjs/js"
	"github.com/gopherjs/websocket/websocketjs"
)

func TestWSInvalidURL(t_ *testing.T) {
	t := testevents.Start(t_, "TestWSInvalidURL", true)
	defer t.Done()

	ws, err := websocketjs.New("blah://blah.example/invalid")
	if err == nil {
		ws.Close()
		t.Fatalf("Got no error, but expected an invalid URL error")
	}
}

func TestWSImmediateClose(t_ *testing.T) {
	t := testevents.Start(t_, "TestWSImmediateClose", true)
	defer t.Done()

	ws, err := websocketjs.New(getWSBaseURL() + "immediate-close")
	if err != nil {
		t.Fatalf("Error opening WebSocket: %s", err)
	}
	defer ws.Close()

	var wg sync.WaitGroup

	ws.AddEventListener("open", false, func(ev *js.Object) {
		t.Logf("WebSocket opened")
	})

	ws.AddEventListener("close", false, func(ev *js.Object) {
		const (
			CloseNormalClosure    = 1000
			CloseNoStatusReceived = 1005 // IE10 hates it when the server closes without sending a close reason
		)
		defer wg.Done()

		closeEventCode := ev.Get("code").Int()

		if closeEventCode != CloseNormalClosure && closeEventCode != CloseNoStatusReceived {
			t.Fatalf("WebSocket close was not clean (code %d)", closeEventCode)
			return
		}
		t.Log("WebSocket closed")
	})
	wg.Add(1)
	wg.Wait()
}
