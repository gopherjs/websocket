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
	"time"

	"github.com/gopherjs/gopherwasm/js"
	"github.com/gopherjs/websocket/websocketjs"
)

// closeError allows a CloseEvent to be used as an error.
type closeError struct {
	js.Value
	Code     int    `js:"code"`
	Reason   string `js:"reason"`
	WasClean bool   `js:"wasClean"`
}

func (e *closeError) Error() string {
	var cleanStmt string
	if e.WasClean {
		cleanStmt = "clean"
	} else {
		cleanStmt = "unclean"
	}
	return fmt.Sprintf("CloseEvent: (%s) (%d) %s", cleanStmt, e.Code, e.Reason)
}

type deadlineErr struct{}

func (e *deadlineErr) Error() string   { return "i/o timeout: deadline reached" }
func (e *deadlineErr) Timeout() bool   { return true }
func (e *deadlineErr) Temporary() bool { return true }

var errDeadlineReached = &deadlineErr{}

// Dial opens a new WebSocket connection. It will block until the connection is
// established or fails to connect.
func Dial(url string) (net.Conn, error) {
	ws, err := websocketjs.New(url)
	if err != nil {
		return nil, err
	}
	conn := &conn{
		WebSocket: ws,
		ch:        make(chan []byte, 1),
	}
	// We need this so that received binary data is in ArrayBufferView format so
	// that it can easily be read.
	conn.SetBinaryType("arraybuffer")

	conn.OnMessage(conn.onMessage)
	conn.OnClose(conn.onClose)

	openCh := make(chan error, 1)

	conn.OnOpen(func() {
		close(openCh)
	})
	//ws.Call("addEventListener", "close", closeHandler, false)

	err, ok := <-openCh
	if ok && err != nil {
		return nil, err
	}

	return conn, nil
}

// conn is a high-level wrapper around WebSocket. It implements net.Conn interface.
type conn struct {
	*websocketjs.WebSocket

	ch      chan []byte
	readBuf *bytes.Reader

	readDeadline time.Time
}

func (c *conn) onMessage(data []byte) {
	c.ch <- data
}

func (c *conn) onClose() {
	// We queue nil to the end so that any messages received prior to
	// closing get handled first.
	c.ch <- nil
}

// handleFrame handles a single frame received from the channel. This is a
// convenience funciton to dedupe code for the multiple deadline cases.
func (c *conn) handleFrame(message []byte, ok bool) ([]byte, error) {
	if !ok { // The channel has been closed
		return nil, io.EOF
	} else if message == nil {
		// See onClose for the explanation about sending a nil item.
		c.Release()
		close(c.ch)
		return nil, io.EOF
	}

	return message, nil
}

// receiveFrame receives one full frame from the WebSocket. It blocks until the
// frame is received.
func (c *conn) receiveFrame(observeDeadline bool) ([]byte, error) {
	var deadlineChan <-chan time.Time // Receiving on a nil channel always blocks indefinitely

	if observeDeadline && !c.readDeadline.IsZero() {
		now := time.Now()
		if now.After(c.readDeadline) {
			select {
			case item, ok := <-c.ch:
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
	case item, ok := <-c.ch:
		return c.handleFrame(item, ok)
	case <-deadlineChan:
		return nil, errDeadlineReached
	}
}

func (c *conn) Read(b []byte) (n int, err error) {
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

	receivedBytes, err := c.receiveFrame(true)
	if err != nil {
		return 0, err
	}

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
	err = c.Send(b)
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
	wsURL, err := url.Parse(c.URL)
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
