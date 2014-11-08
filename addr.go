package websocket

import "net/url"

// Addr represents the address of a WebSocket connection.
type Addr struct {
	*url.URL
}

// Network returns the network type for a WebSocket, "websocket".
func (addr *Addr) Network() string { return "websocket" }
