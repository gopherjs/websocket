// Copyright 2014-2015 GopherJS Team. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be found
// in the LICENSE file.

package websocket

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"net/url"
	"sync/atomic"
	"syscall/js"
	"time"

	"github.com/gopherjs/websocket/websocketjs"
)

func beginHandlerOpen(ch chan error, removeHandlers func()) func(ev js.Value) {
	return func(ev js.Value) {
		removeHandlers()
		close(ch)
	}
}

// closeError allows a CloseEvent to be used as an error.
type closeError struct {
	js.Value
}

func (e *closeError) Error() string {
	var cleanStmt string
	if e.Get("wasClean").Bool() {
		cleanStmt = "clean"
	} else {
		cleanStmt = "unclean"
	}
	return fmt.Sprintf("CloseEvent: (%s) (%d) %s", cleanStmt, e.Get("code").Int(), e.Get("reason").String())
}

func beginHandlerClose(ch chan error, removeHandlers func()) func(ev js.Value) {
	return func(ev js.Value) {
		removeHandlers()
		ch <- &closeError{Value: ev}
		close(ch)
	}
}

type deadlineErr struct{}

func (e *deadlineErr) Error() string   { return "i/o timeout: deadline reached" }
func (e *deadlineErr) Timeout() bool   { return true }
func (e *deadlineErr) Temporary() bool { return true }

var errDeadlineReached = &deadlineErr{}

// TODO(nightexcessive): Add a Dial function that allows a deadline to be
// specified.

// Dial opens a new WebSocket connection. It will block until the connection is
// established or fails to connect.
func Dial(url string) (net.Conn, error) {
	ws, err := websocketjs.New(url)
	if err != nil {
		return nil, err
	}
	conn := &conn{
		WebSocket: ws,
	}
	conn.isClosed = new(uint32)
	*conn.isClosed = 0
	conn.initialize()

	openCh := make(chan error, 1)

	var (
		openHandler  js.Callback
		closeHandler js.Callback
	)

	// Handlers need to be removed to prevent a panic when the WebSocket closes
	// immediately and fires both open and close before they can be removed.
	// This way, handlers are removed before the channel is closed.
	removeHandlers := func() {
		ws.RemoveEventListener("open", openHandler)
		ws.RemoveEventListener("close", closeHandler)
		openHandler.Release()
		closeHandler.Release()
	}

	// We have to use variables for the functions so that we can remove the
	// event handlers afterwards.
	openHandler = js.NewEventCallback(0, beginHandlerOpen(openCh, removeHandlers))
	closeHandler = js.NewEventCallback(0, beginHandlerClose(openCh, removeHandlers))

	ws.AddEventListener("open", openHandler)
	ws.AddEventListener("close", closeHandler)

	err, ok := <-openCh
	if ok && err != nil {
		return nil, err
	}

	return conn, nil
}

// conn is a high-level wrapper around WebSocket. It implements net.Conn interface.
type conn struct {
	*websocketjs.WebSocket

	isClosed *uint32

	writeCh chan<- *messageEvent
	readCh  <-chan *messageEvent
	readBuf *bytes.Reader

	onMessageCallback js.Callback
	onCloseCallback   js.Callback

	readDeadline time.Time
}

type messageEvent struct {
	js.Value
	// Data js.Value `js:"data"`
}

func (c *conn) onMessage(event js.Value) {
	c.writeCh <- &messageEvent{Value: event}
}

func (c *conn) onClose(event js.Value) {
	// We queue nil to the end so that any messages received prior to
	// closing get handled first.
	swapped := atomic.CompareAndSwapUint32(c.isClosed, 0, 1)
	if swapped {
		close(c.writeCh)
		c.RemoveEventListener("message", c.onMessageCallback)
		c.onMessageCallback.Release()
		c.RemoveEventListener("close", c.onCloseCallback)
		c.onCloseCallback.Release()
	}
}

func (c *conn) Close() error {
	err := c.WebSocket.Close()
	c.onClose(js.Null())
	return err
}

// initialize adds all of the event handlers necessary for a conn to function.
// It should never be called more than once and is already called if Dial was
// used to create the conn.
func (c *conn) initialize() {
	writeChan := make(chan *messageEvent)
	readChan := make(chan *messageEvent)

	c.writeCh = writeChan
	c.readCh = readChan

	go c.bufferMessageEvents(writeChan, readChan)

	// We need this so that received binary data is in ArrayBufferView format so
	// that it can easily be read.
	c.Set("binaryType", "arraybuffer")

	c.onMessageCallback = js.NewEventCallback(0, c.onMessage)
	c.onCloseCallback = js.NewEventCallback(0, c.onClose)

	c.AddEventListener("message", c.onMessageCallback)
	c.AddEventListener("close", c.onCloseCallback)
}

func (c *conn) bufferMessageEvents(write chan *messageEvent, read chan *messageEvent) {
	queue := make([]*messageEvent, 0, 16)

	getReadChan := func() chan *messageEvent {
		if len(queue) == 0 {
			return nil
		}

		return read
	}

	getQueuedEvent := func() *messageEvent {
		if len(queue) == 0 {
			return nil
		}

		return queue[0]
	}

	for len(queue) > 0 || write != nil {
		select {
		case newEvent, ok := <-write:
			if !ok {
				write = nil
			} else {
				queue = append(queue, newEvent)
			}
		case getReadChan() <- getQueuedEvent():
			queue = queue[1:]
		}
	}

	close(read)
}

// handleFrame handles a single frame received from the channel. This is a
// convenience funciton to dedupe code for the multiple deadline cases.
func (c *conn) handleFrame(message *messageEvent, ok bool) (*messageEvent, error) {
	if !ok { // The channel has been closed
		return nil, io.EOF
	}

	return message, nil
}

// receiveFrame receives one full frame from the WebSocket. It blocks until the
// frame is received.
func (c *conn) receiveFrame(observeDeadline bool) (*messageEvent, error) {
	var deadlineChan <-chan time.Time // Receiving on a nil channel always blocks indefinitely

	if observeDeadline && !c.readDeadline.IsZero() {
		now := time.Now()
		if now.After(c.readDeadline) {
			select {
			case item, ok := <-c.readCh:
				return c.handleFrame(item, ok)
			default:
				return nil, errDeadlineReached
			}
		}

		timer := time.NewTimer(c.readDeadline.Sub(now))
		defer timer.Stop()

		deadlineChan = timer.C
	}

	select {
	case item, ok := <-c.readCh:
		return c.handleFrame(item, ok)
	case <-deadlineChan:
		return nil, errDeadlineReached
	}
}

func getFrameData(obj js.Value) []byte {
	// Check if it's an array buffer. If so, convert it to a Go byte slice.
	if obj.InstanceOf(js.Global().Get("ArrayBuffer")) {
		uint8Array := js.Global().Get("Uint8Array").New(obj)
		data := make([]byte, uint8Array.Length())
		for i, arrayLen := 0, uint8Array.Length(); i < arrayLen; i++ {
			data[i] = byte(uint8Array.Index(i).Int())
		}
		return data
	}
	return []byte(obj.String())
}

func (c *conn) Read(b []byte) (n int, err error) {
	// Fast path: Don't bother trying to read if given a 0-byte buffer to read into
	if len(b) == 0 {
		return 0, nil
	}

	if c.readBuf != nil {
		n, err = c.readBuf.Read(b)
		if err == io.EOF {
			c.readBuf = nil
			err = nil
		}
		// If we read nothing from the buffer, continue to trying to receive.
		// This saves us when the last Read call emptied the buffer and this
		// call triggers the EOF. There's probably a better way of doing this,
		// but I'm really tired.
		if n > 0 {
			return
		}
	}

	frame, err := c.receiveFrame(true)
	if err != nil {
		return 0, err
	}

	receivedBytes := getFrameData(frame.Get("data"))

	n = copy(b, receivedBytes)
	// Fast path: The entire frame's contents have been copied into b.
	if n >= len(receivedBytes) {
		return
	}

	c.readBuf = bytes.NewReader(receivedBytes[n:])
	return
}

// Write writes the contents of b to the WebSocket using a binary opcode.
func (c *conn) Write(b []byte) (n int, err error) {
	// []byte is converted to an Uint8Array by GopherJS, which fullfils the
	// ArrayBufferView definition.
	byteArray := js.TypedArrayOf(b)
	defer byteArray.Release()
	err = c.Send(byteArray.Value)
	if err != nil {
		return 0, err
	}
	return len(b), nil
}

// LocalAddr would typically return the local network address, but due to
// limitations in the JavaScript API, it is unable to. Calling this method will
// cause a panic.
func (c *conn) LocalAddr() net.Addr {
	// BUG(nightexcessive): conn.LocalAddr() panics because the underlying
	// JavaScript API has no way of figuring out the local address.

	// TODO(nightexcessive): Find a more graceful way to handle this
	panic("we are unable to implement websocket.conn.LocalAddr() due to limitations in the underlying JavaScript API")
}

// RemoteAddr returns the remote network address, based on
// websocket.WebSocket.URL.
func (c *conn) RemoteAddr() net.Addr {
	wsURL, err := url.Parse(c.Get("url").String())
	if err != nil {
		// TODO(nightexcessive): Should we be panicking for this?
		panic(err)
	}
	return &addr{wsURL}
}

// SetDeadline sets the read and write deadlines associated with the connection.
// It is equivalent to calling both SetReadDeadline and SetWriteDeadline.
//
// A zero value for t means that I/O operations will not time out.
func (c *conn) SetDeadline(t time.Time) error {
	c.readDeadline = t
	return nil
}

// SetReadDeadline sets the deadline for future Read calls. A zero value for t
// means Read will not time out.
func (c *conn) SetReadDeadline(t time.Time) error {
	c.readDeadline = t
	return nil
}

// SetWriteDeadline sets the deadline for future Write calls. Because our writes
// do not block, this function is a no-op.
func (c *conn) SetWriteDeadline(t time.Time) error {
	return nil
}
