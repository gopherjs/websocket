//go:generate gopherjs build -m index.go

package websocket_test

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"sync"
	"syscall/js"
	"testing"
	"time"

	"github.com/LinearZoetrope/testevents"
	"github.com/gopherjs/websocket"
	"github.com/gopherjs/websocket/websocketjs"
)

func testStarted(ev testevents.Event) {
	document := js.Global().Get("document")
	body := document.Get("body")

	outsideElement := document.Call("createElement", "div")
	outsideElement.Set("id", ev.Name)

	nameElement := document.Call("createElement", "span")
	nameElement.Set("textContent", fmt.Sprintf("%s:", ev.Name))
	outsideElement.Call("appendChild", nameElement)

	nbspElement := document.Call("createElement", "span")
	nbspElement.Set("innerHTML", "&nbsp;")
	outsideElement.Call("appendChild", nbspElement)

	body.Call("appendChild", outsideElement)
}

func testPassed(ev testevents.Event) {
	document := js.Global().Get("document")
	outsideElement := document.Call("getElementById", ev.Name)

	statusElement := document.Call("createElement", "span")
	statusElement.Set("textContent", "PASSED")
	outsideElement.Call("appendChild", statusElement)
}

func testFailed(ev testevents.Event) {
	document := js.Global().Get("document")
	outsideElement := document.Call("getElementById", ev.Name)

	statusElement := document.Call("createElement", "span")
	statusElement.Set("textContent", "FAILED")
	outsideElement.Call("appendChild", statusElement)
}

func init() {
	testevents.Register(testevents.TestStarted, testevents.EventListener{testStarted, int(testevents.TestStarted)})
	testevents.Register(testevents.TestPassed, testevents.EventListener{testPassed, int(testevents.TestPassed)})
	testevents.Register(testevents.TestFailed, testevents.EventListener{testFailed, int(testevents.TestFailed)})

	flag.Set("test.v", "true")
	flag.Set("test.parallel", "4")
}

func getWSBaseURL() string {
	location := js.Global().Get("window").Get("document").Get("location")

	wsProtocol := "ws"
	if location.Get("protocol").String() == "https:" {
		wsProtocol = "wss"
	}

	return fmt.Sprintf("%s://%s:%s/ws/", wsProtocol, location.Get("hostname").String(), location.Get("port").String())
}

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

func TestConnImmediateClose(t_ *testing.T) {
	t := testevents.Start(t_, "TestConnImmediateClose", true)
	defer t.Done()

	ws, err := websocket.Dial(getWSBaseURL() + "immediate-close")
	if err != nil {
		t.Fatalf("Error opening WebSocket: %s", err)
	}
	defer ws.Close()

	t.Log("WebSocket opened")

	_, err = ws.Read(nil)
	if err == io.EOF {
		t.Log("Received EOF")
	} else if err != nil {
		t.Fatalf("Unexpected error in second read: %s", err)
	} else {
		t.Fatalf("Expected EOF in second read, got no error")
	}
}

func TestConnFailedOpen(t_ *testing.T) {
	t := testevents.Start(t_, "TestConnFailedOpen", true)
	defer t.Done()

	ws, err := websocket.Dial(getWSBaseURL() + "404-not-found")
	if err == nil {
		ws.Close()
		t.Fatalf("Got no error, but expected an error in opening the WebSocket.")
	}

	t.Logf("WebSocket failed to open: %s", err)
}

func TestConnBinaryRead(t_ *testing.T) {
	t := testevents.Start(t_, "TestConnBinaryRead", true)
	defer t.Done()

	ws, err := websocket.Dial(getWSBaseURL() + "binary-static")
	if err != nil {
		t.Fatalf("Error opening WebSocket: %s", err)
	}
	defer ws.Close()

	t.Logf("WebSocket opened")

	var expectedData = []byte{0x00, 0x01, 0x02, 0x03, 0x04}

	receivedData := make([]byte, len(expectedData))
	n, err := ws.Read(receivedData)
	if err != nil {
		t.Fatalf("Error in first read: %s", err)
		return
	}
	receivedData = receivedData[:n]

	if !bytes.Equal(receivedData, expectedData) {
		t.Fatalf("Received data did not match expected data. Got % x, expected % x.", receivedData, expectedData)
	} else {
		t.Logf("Received data: % x", receivedData)
	}

	_, err = ws.Read(receivedData)
	if err == io.EOF {
		t.Logf("Received EOF")
	} else if err != nil {
		t.Fatalf("Unexpected error in second read: %s", err)
	} else {
		t.Fatalf("Expected EOF in second read, got no error")
	}
}

func TestConnMultiFrameRead(t_ *testing.T) {
	t := testevents.Start(t_, "TestConnMultiFrameRead", true)
	defer t.Done()

	ws, err := websocket.Dial(getWSBaseURL() + "multiframe-static")
	if err != nil {
		t.Fatalf("Error opening WebSocket: %s", err)
	}
	defer ws.Close()

	t.Logf("WebSocket opened")

	var expectedData = []byte{0x00, 0x01, 0x02, 0x03, 0x04}

	receivedData := make([]byte, len(expectedData))
	n, err := io.ReadAtLeast(ws, receivedData, len(expectedData))
	if err != nil {
		t.Fatalf("Error in read: %s", err)
		return
	}
	receivedData = receivedData[:n]

	if !bytes.Equal(receivedData, expectedData) {
		t.Fatalf("Received data did not match expected data. Got % x, expected % x.", receivedData, expectedData)
	} else {
		t.Logf("Received data: % x", receivedData)
	}

	_, err = ws.Read(receivedData)
	if err == io.EOF {
		t.Logf("Received EOF")
	} else if err != nil {
		t.Fatalf("Unexpected error in second read: %s", err)
	} else {
		t.Fatalf("Expected EOF in second read, got no error")
	}
}

func TestConn1MBRead(t_ *testing.T) {
	t := testevents.Start(t_, "TestConn1MBRead", true)
	defer t.Done()

	ws, err := websocket.Dial(getWSBaseURL() + "random-1mb")
	if err != nil {
		t.Fatalf("Error opening WebSocket: %s", err)
	}
	defer ws.Close()

	bytesRead := 0
	data := make([]byte, 1024)
	for i := 0; i < 1024; i++ {
		n, err := io.ReadAtLeast(ws, data, len(data))
		if err != nil {
			t.Fatalf("Error reading 1024 bytes: %s", err)
		}
		bytesRead = bytesRead + n
	}

	if bytesRead != 1024*1024 {
		t.Fatalf("Read %d bytes; expected %d bytes", bytesRead, 1024*1024)
	}
	t.Logf("%d bytes successfuly read", bytesRead)
}

func TestWSTimeout(t_ *testing.T) {
	t := testevents.Start(t_, "TestWSTimeout", true)
	defer t.Done()

	ws, err := websocket.Dial(getWSBaseURL() + "wait-30s")
	if err != nil {
		t.Fatalf("Error opening WebSocket: %s", err)
	}
	defer ws.Close()

	t.Logf("WebSocket opened")

	start := time.Now()
	timeoutTime := time.Now().Add(1 * time.Second)
	ws.SetReadDeadline(timeoutTime)

	_, err = ws.Read(nil)
	if err != nil && err.Error() == "i/o timeout: deadline reached" {
		totalTime := time.Now().Sub(start)
		if time.Now().Before(timeoutTime) {
			t.Fatalf("Timeout was too short: Received timeout after %s", totalTime)
		}
		t.Logf("Received timeout after %s", totalTime)
	} else if err != nil {
		t.Fatalf("Unexpected error in read: %s", err)
	} else {
		t.Fatalf("Expected timeout in read, got no error")
	}
}
