package websocket

import (
	"net/url"
	"time"
)

// Conn is a high-level wrapper around WebSocket. It is intended to
// provide a net.TCPConn-like interface.
type Conn struct {
	*WebSocket
}

func (c *Conn) Read(b []byte) (n int, err error) {
	// TODO(nightexcessive): Implement
	panic("not yet implemeneted")
}

func (c *Conn) Write(b []byte) (n int, err error) {
	// TODO(nightexcessive): Implement
	panic("not yet implemeneted")
}

// BUG(nightexcessive): We can't return net.Addr from Conn.LocalAddr and
// Conn.RemoteAddr because net.init() causes a panic due to attempts to make
// syscalls. See: https://github.com/gopherjs/gopherjs/issues/123

// LocalAddr would typically return the local network address, but due to
// limitations in the JavaScript API, it is unable to. Calling this method will
// cause a panic.
func (c *Conn) LocalAddr() *Addr {
	// BUG(nightexcessive): Conn.LocalAddr() panics because the underlying
	// JavaScript API has no way of figuring out the local address.

	// TODO(nightexcessive): Find a more graceful way to handle this
	panic("we are unable to implement websocket.Conn.LocalAddr() due to limitations in the underlying JavaScript API")
}

// RemoteAddr returns the remote network address, based on
// websocket.WebSocket.URL.
func (c *Conn) RemoteAddr() *Addr {
	wsURL, err := url.Parse(c.URL)
	if err != nil {
		// TODO(nightexcessive): Should we be panicking for this?
		panic(err)
	}
	return &Addr{wsURL}
}

// SetDeadline implements the net.Conn.SetDeadline method.
func (c *Conn) SetDeadline(t time.Time) error {
	// TODO(nightexcessive): Implement
	panic("not yet implemeneted")
}

// SetReadDeadline implements the net.Conn.SetReadDeadline method.
func (c *Conn) SetReadDeadline(t time.Time) error {
	// TODO(nightexcessive): Implement
	panic("not yet implemeneted")
}

// SetWriteDeadline implements the net.Conn.SetWriteDeadline method.
func (c *Conn) SetWriteDeadline(t time.Time) error {
	// TODO(nightexcessive): Implement
	panic("not yet implemeneted")
}
