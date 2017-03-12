// Copyright 2014-2015 GopherJS Team. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be found
// in the LICENSE file.

/*
Package websocket provides high-level bindings for the browser's WebSocket API.

These bindings offer a Dial function that returns a regular net.Conn.
They can be used similar to net package. For example:

	c, err := websocket.Dial("ws://localhost/socket") // Blocks until connection is established
	if err != nil { ... }

	buf := make([]byte, 1024)
	n, err = c.Read(buf) // Blocks until a WebSocket frame is received
	if err != nil { ... }
	doSomethingWithData(buf[:n])

	_, err = c.Write([]byte("Hello!"))
	if err != nil { ... }

	err = c.Close()
	if err != nil { ... }
*/
package websocket
