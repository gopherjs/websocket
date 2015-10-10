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

	"github.com/gopherjs/gopherjs/js"
	"honnef.co/go/js/console"
	"honnef.co/go/js/dom"
)

func beginHandlerOpen(ch chan error, removeHandlers func()) func(ev *js.Object) {
	return func(ev *js.Object) {
		removeHandlers()
		close(ch)
	}
}

// closeErrorMap maps Close frame status codes to their string explanation
// See http://tools.ietf.org/html/rfc6455#section-7.4.1
var closeErrorMap = map[int]string{
	1000: "Normal closure, meaning that the purpose for which the connection was established has been fulfilled.",
	1001: "An endpoint is \"going away\", such as a server going down or a browser having navigated away from a page.",
	1002: "An endpoint is terminating the connection due to a protocol error",
	1003: "An endpoint is terminating the connection because it has received a type of data it cannot accept (e.g., an endpoint that understands only text data MAY send this if it receives a binary message).",
	1004: "Reserved. The specific meaning might be defined in the future.",
	1005: "No status code was actually present.",
	1006: "The connection was closed abnormally, e.g., without sending or receiving a Close control frame",
	1007: "An endpoint is terminating the connection because it has received data within a message that was not consistent with the type of the message (e.g., non-UTF-8 [http://tools.ietf.org/html/rfc3629] data within a text message).",
	1008: "An endpoint is terminating the connection because it has received a message that \"violates its policy\". This reason is given either if there is no other sutible reason, or if there is a need to hide specific details about the policy.",
	1009: "An endpoint is terminating the connection because it has received a message that is too big for it to process.",
	// Note that this status code is not used by the server, because it can fail the WebSocket handshake instead
	1010: "An endpoint (client) is terminating the connection because it has expected the server to negotiate one or more extension, but the server didn't return them in the response message of the WebSocket handshake.",
	1011: "A server is terminating the connection because it encountered an unexpected condition that prevented it from fulfilling the request.",
	1015: "The connection was closed due to a failure to perform a TLS handshake (e.g., the server certificate can't be verified).",
}

// closeError allows a CloseEvent to be used as an error.
type closeError struct {
	*dom.CloseEvent
}

func (e *closeError) Error() string {
	var (
		cleanStmt string
		reason    string // cant override the event's reason
		ok        bool
	)
	if e.WasClean {
		cleanStmt = "clean"
	} else {
		cleanStmt = "unclean"

		if e.Reason != "" {
			reason = e.Reason
		} else {
			reason, ok = closeErrorMap[e.Code]
			if !ok {
				reason = "Unknown reason"
			}
		}

	}

	return fmt.Sprintf("CloseEvent: (%s) (%d) %s", cleanStmt, e.Code, reason)
}

func beginHandlerClose(ch chan error, removeHandlers func()) func(ev *js.Object) {
	return func(ev *js.Object) {
		removeHandlers()
		go func() {
			ce := dom.WrapEvent(ev).(*dom.CloseEvent)
			ch <- &closeError{CloseEvent: ce}
			close(ch)
		}()
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
func Dial(url string) (*Conn, error) {
	ws, err := New(url)
	if err != nil {
		return nil, err
	}
	conn := &Conn{
		WebSocket: ws,
		ch:        make(chan *dom.MessageEvent, 1),
	}
	conn.initialize()
	conn.Reader, conn.pw = io.Pipe()

	openCh := make(chan error, 1)

	var (
		openHandler  func(ev *js.Object)
		closeHandler func(ev *js.Object)
	)

	// Handlers need to be removed to prevent a panic when the WebSocket closes
	// immediately and fires both open and close before they can be removed.
	// This way, handlers are removed before the channel is closed.
	removeHandlers := func() {
		ws.RemoveEventListener("open", false, openHandler)
		ws.RemoveEventListener("close", false, closeHandler)
	}

	// We have to use variables for the functions so that we can remove the
	// event handlers afterwards.
	openHandler = beginHandlerOpen(openCh, removeHandlers)
	closeHandler = beginHandlerClose(openCh, removeHandlers)

	ws.AddEventListener("open", false, openHandler)
	ws.AddEventListener("close", false, closeHandler)

	err, ok := <-openCh
	if ok && err != nil {
		return nil, err
	}

	go conn.receiveLoop()

	return conn, nil
}

// Conn is a high-level wrapper around WebSocket. It is intended to satisfy the
// net.Conn interface.
//
// To create a Conn, use Dial. Instantiating Conn without Dial will not work.
type Conn struct {
	*WebSocket

	ch      chan *dom.MessageEvent
	readBuf *bytes.Reader

	readDeadline time.Time

	io.Reader
	pw       *io.PipeWriter
	closeErr *closeError
}

func (c *Conn) onMessage(event *js.Object) {
	go func() {
		c.ch <- dom.WrapEvent(event).(*dom.MessageEvent)
	}()
}

func (c *Conn) onClose(event *js.Object) {
	go func() {
		if ce, ok := dom.WrapEvent(event).(*dom.CloseEvent); ok {
			c.closeErr = &closeError{CloseEvent: ce}
			console.Error(c.closeErr.Error())
		}

		// We queue nil to the end so that any messages received prior to
		// closing get handled first.
		c.ch <- nil
	}()
}

// initialize adds all of the event handlers necessary for a Conn to function.
// It should never be called more than once and is already called if Dial was
// used to create the Conn.
func (c *Conn) initialize() {
	// We need this so that received binary data is in ArrayBufferView format so
	// that it can easily be read.
	c.BinaryType = "arraybuffer"

	c.AddEventListener("message", false, c.onMessage)
	c.AddEventListener("close", false, c.onClose)
}

// handleFrame handles a single frame received from the channel. This is a
// convenience funciton to dedupe code for the multiple deadline cases.
func (c *Conn) handleFrame(item *dom.MessageEvent, ok bool) (*dom.MessageEvent, error) {
	if !ok { // The channel has been closed
		if c.closeErr == nil {
			return nil, io.EOF

		}
		return nil, c.closeErr
	} else if item == nil {
		// See onClose for the explanation about sending a nil item.
		close(c.ch)
		return nil, io.EOF
	}

	return item, nil
}

// receiveLoop fill's the writer end of the Conn's io.Pipe with data frames
// runs in its own goroutine
func (c *Conn) receiveLoop() {

	for {
		frame, err := c.receiveFrame(true)
		if err != nil {
			if err2 := c.pw.CloseWithError(err); err2 != nil {
				console.Error("CloseWithError failed", err2)
			}
			return
		}

		receivedBytes := getFrameData(frame.Data)

		_, err = io.Copy(c.pw, bytes.NewReader(receivedBytes))
		if err != nil {
			if err2 := c.pw.CloseWithError(err); err2 != nil {
				console.Error("CloseWithError failed", err2)
			}
			return
		}
	}
}

// receiveFrame receives one full frame from the WebSocket. It blocks until the
// frame is received.
func (c *Conn) receiveFrame(observeDeadline bool) (*dom.MessageEvent, error) {
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

func getFrameData(obj *js.Object) []byte {
	// Check if it's an array buffer. If so, convert it to a Go byte slice.
	if constructor := obj.Get("constructor"); constructor == js.Global.Get("ArrayBuffer") {
		int8Array := js.Global.Get("Uint8Array").New(obj)
		return int8Array.Interface().([]byte)
	}

	return []byte(obj.String())
}

// Write writes the contents of b to the WebSocket using a binary opcode.
func (c *Conn) Write(b []byte) (n int, err error) {
	if c.closeErr != nil {
		return 0, c.closeErr
	}
	// []byte is converted to an (U)Int8Array by GopherJS, which fullfils the
	// ArrayBufferView definition.
	err = c.Send(b)
	if err != nil {
		return
	}
	n = len(b)
	return
}

// WriteString writes the contents of s to the WebSocket using a text frame
// opcode.
func (c *Conn) WriteString(s string) (n int, err error) {
	err = c.Send(s)
	if err != nil {
		return
	}
	n = len(s)
	return
}

// LocalAddr would typically return the local network address, but due to
// limitations in the JavaScript API, it is unable to. Calling this method will
// cause a panic.
func (c *Conn) LocalAddr() net.Addr {
	// BUG(nightexcessive): Conn.LocalAddr() panics because the underlying
	// JavaScript API has no way of figuring out the local address.

	// TODO(nightexcessive): Find a more graceful way to handle this
	panic("we are unable to implement websocket.Conn.LocalAddr() due to limitations in the underlying JavaScript API")
}

// RemoteAddr returns the remote network address, based on
// websocket.WebSocket.URL.
func (c *Conn) RemoteAddr() net.Addr {
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
func (c *Conn) SetDeadline(t time.Time) error {
	c.readDeadline = t
	return nil
}

// SetReadDeadline sets the deadline for future Read calls. A zero value for t
// means Read will not time out.
func (c *Conn) SetReadDeadline(t time.Time) error {
	c.readDeadline = t
	return nil
}

// SetWriteDeadline sets the deadline for future Write calls. Because our writes
// do not block, this function is a no-op.
func (c *Conn) SetWriteDeadline(t time.Time) error {
	return nil
}
