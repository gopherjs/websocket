//go:generate gopherjs build -m index.go

package main

import (
	"bytes"
	"fmt"
	"io"
	"time"

	"github.com/gopherjs/gopherjs/js"
	"github.com/gopherjs/websocket"
	"github.com/gopherjs/websocket/websocketjs"
	"github.com/rusco/qunit"
)

func getWSBaseURL() string {
	document := js.Global.Get("window").Get("document")
	location := document.Get("location")

	wsProtocol := "ws"
	if location.Get("protocol").String() == "https:" {
		wsProtocol = "wss"
	}

	return fmt.Sprintf("%s://%s:%s/ws/", wsProtocol, location.Get("hostname"), location.Get("port"))
}

func main() {
	wsBaseURL := getWSBaseURL()

	qunit.Module("websocketjs.WebSocket")
	qunit.Test("Invalid URL", func(assert qunit.QUnitAssert) {
		qunit.Expect(1)

		ws, err := websocketjs.New("blah://blah.example/invalid")
		if err == nil {
			ws.Close()
			assert.Ok(false, "Got no error, but expected an invalid URL error")
			return
		}

		assert.Ok(true, fmt.Sprintf("Received an error: %s", err))
	})
	qunit.AsyncTest("Immediate close", func() interface{} {
		qunit.Expect(2)

		ws, err := websocketjs.New(wsBaseURL + "immediate-close")
		if err != nil {
			qunit.Ok(false, fmt.Sprintf("Error opening WebSocket: %s", err))
			qunit.Start()
			return nil
		}

		ws.AddEventListener("open", false, func(ev *js.Object) {
			qunit.Ok(true, "WebSocket opened")
		})

		ws.AddEventListener("close", false, func(ev *js.Object) {
			const (
				CloseNormalClosure    = 1000
				CloseNoStatusReceived = 1005 // IE10 hates it when the server closes without sending a close reason
			)

			closeEventCode := ev.Get("code").Int()

			if closeEventCode != CloseNormalClosure && closeEventCode != CloseNoStatusReceived {
				qunit.Ok(false, fmt.Sprintf("WebSocket close was not clean (code %d)", closeEventCode))
				qunit.Start()
				return
			}
			qunit.Ok(true, "WebSocket closed")
			qunit.Start()
		})

		return nil
	})

	qunit.Module("websocket.Conn")
	qunit.AsyncTest("Immediate close", func() interface{} {
		go func() {
			defer qunit.Start()

			ws, err := websocket.Dial(wsBaseURL + "immediate-close")
			if err != nil {
				qunit.Ok(false, fmt.Sprintf("Error opening WebSocket: %s", err))
				return
			}

			qunit.Ok(true, "WebSocket opened")

			_, err = ws.Read(nil)
			if err == io.EOF {
				qunit.Ok(true, "Received EOF")
			} else if err != nil {
				qunit.Ok(false, fmt.Sprintf("Unexpected error in second read: %s", err))
			} else {
				qunit.Ok(false, "Expected EOF in second read, got no error")
			}
		}()

		return nil
	})
	qunit.AsyncTest("Failed open", func() interface{} {
		go func() {
			defer qunit.Start()

			ws, err := websocket.Dial(wsBaseURL + "404-not-found")
			if err == nil {
				ws.Close()
				qunit.Ok(false, "Got no error, but expected an error in opening the WebSocket.")
				return
			}

			qunit.Ok(true, fmt.Sprintf("WebSocket failed to open: %s", err))
		}()

		return nil
	})
	qunit.AsyncTest("Binary read", func() interface{} {
		qunit.Expect(3)

		go func() {
			defer qunit.Start()

			ws, err := websocket.Dial(wsBaseURL + "binary-static")
			if err != nil {
				qunit.Ok(false, fmt.Sprintf("Error opening WebSocket: %s", err))
				return
			}

			qunit.Ok(true, "WebSocket opened")

			var expectedData = []byte{0x00, 0x01, 0x02, 0x03, 0x04}

			receivedData := make([]byte, len(expectedData))
			n, err := ws.Read(receivedData)
			if err != nil {
				qunit.Ok(false, fmt.Sprintf("Error in first read: %s", err))
				return
			}
			receivedData = receivedData[:n]

			if !bytes.Equal(receivedData, expectedData) {
				qunit.Ok(false, fmt.Sprintf("Received data did not match expected data. Got % x, expected % x.", receivedData, expectedData))
			} else {
				qunit.Ok(true, fmt.Sprintf("Received data: % x", receivedData))
			}

			_, err = ws.Read(receivedData)
			if err == io.EOF {
				qunit.Ok(true, "Received EOF")
			} else if err != nil {
				qunit.Ok(false, fmt.Sprintf("Unexpected error in second read: %s", err))
			} else {
				qunit.Ok(false, "Expected EOF in second read, got no error")
			}
		}()

		return nil
	})
	qunit.AsyncTest("Timeout", func() interface{} {
		qunit.Expect(2)

		go func() {
			defer qunit.Start()

			ws, err := websocket.Dial(wsBaseURL + "wait-30s")
			if err != nil {
				qunit.Ok(false, fmt.Sprintf("Error opening WebSocket: %s", err))
				return
			}

			qunit.Ok(true, "WebSocket opened")

			start := time.Now()
			ws.SetReadDeadline(start.Add(1 * time.Second))

			_, err = ws.Read(nil)
			if err != nil && err.Error() == "i/o timeout: deadline reached" {
				totalTime := time.Now().Sub(start)
				if totalTime < 750*time.Millisecond {
					qunit.Ok(false, fmt.Sprintf("Timeout was too short: Received timeout after %s", totalTime))
					return
				}
				qunit.Ok(true, fmt.Sprintf("Received timeout after %s", totalTime))
			} else if err != nil {
				qunit.Ok(false, fmt.Sprintf("Unexpected error in read: %s", err))
			} else {
				qunit.Ok(false, "Expected timeout in read, got no error")
			}
		}()

		return nil
	})
}
