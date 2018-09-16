package websocket_test

import (
	"sync"
	"syscall/js"
	"testing"

	"github.com/LinearZoetrope/testevents"
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

	var (
		openCallback  js.Callback
		closeCallback js.Callback
	)

	openCallback = js.NewEventCallback(0, func(ev js.Value) {
		defer ws.RemoveEventListener("open", openCallback)

		t.Logf("WebSocket opened")
	})
	defer openCallback.Release()
	ws.AddEventListener("open", openCallback)

	closeCallback = js.NewEventCallback(0, func(ev js.Value) {
		defer wg.Done()
		defer ws.RemoveEventListener("close", closeCallback)

		const (
			CloseNormalClosure    = 1000
			CloseNoStatusReceived = 1005 // IE10 hates it when the server closes without sending a close reason
		)

		closeEventCode := ev.Get("code").Int()

		if closeEventCode != CloseNormalClosure && closeEventCode != CloseNoStatusReceived {
			t.Fatalf("WebSocket close was not clean (code %d)", closeEventCode)
		}
		t.Logf("WebSocket closed")
	})
	defer closeCallback.Release()
	ws.AddEventListener("close", closeCallback)
	wg.Add(1)

	wg.Wait()
}
