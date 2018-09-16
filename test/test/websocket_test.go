package websocket_test

import (
	"bytes"
	"crypto/rand"
	"io"
	"testing"
	"time"

	"github.com/LinearZoetrope/testevents"
	"github.com/gopherjs/websocket"
)

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

func TestConnBinaryEcho(t_ *testing.T) {
	t := testevents.Start(t_, "TestConnBinaryEcho", true)
	defer t.Done()

	data := make([]byte, 1024*1024)

	totalN := 0
	for totalN < len(data) {
		sliceEnd := totalN + 65535
		if sliceEnd > len(data) {
			sliceEnd = len(data)
		}
		n, err := rand.Read(data[totalN:sliceEnd])
		if err != nil {
			t.Fatalf("Error in creating data: %s", err)
		}
		totalN = totalN + n
	}

	data = data[:totalN]

	t.Logf("Created %d bytes to send", len(data))

	ws, err := websocket.Dial(getWSBaseURL() + "echo")
	if err != nil {
		t.Fatalf("Error opening WebSocket: %s", err)
	}
	defer ws.Close()

	t.Logf("WebSocket opened")

	byteReader := bytes.NewReader(data)
	nSent, err := io.Copy(ws, byteReader)
	if err != nil {
		t.Fatalf("Error sending data: %s", err)
	}

	t.Logf("Sent %d bytes", nSent)

	receivedData := make([]byte, len(data))
	n, err := io.ReadAtLeast(ws, receivedData, len(receivedData))
	if err != nil {
		t.Fatalf("Error in read: %s", err)
	}
	receivedData = receivedData[:n]

	t.Logf("Received %d bytes", n)

	if !bytes.Equal(receivedData, data) {
		t.Fatalf("Received data did not match expected data.")
	} else {
		t.Logf("Received correct data")
	}

	receivedData = receivedData[:256]

	// Check for extra data
	ws.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
	n, err = ws.Read(receivedData)
	if n != 0 {
		t.Fatalf("Extra data was received")
	} else if err != nil && err.Error() == "i/o timeout: deadline reached" {
		t.Logf("No extra data received")
	} else if err != nil {
		t.Fatalf("Error checking for extra data: %s", err)
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

func TestConnTimeout(t_ *testing.T) {
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
