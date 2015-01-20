// Copyright 2014-2015 GopherJS Team. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be found
// in the LICENSE file.

package websocket

import "net/url"

// addr represents the address of a WebSocket connection.
type addr struct {
	*url.URL
}

// Network returns the network type for a WebSocket, "websocket".
func (addr *addr) Network() string { return "websocket" }
