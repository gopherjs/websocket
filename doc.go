// Copyright 2014-2015 GopherJS Team. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be found
// in the LICENSE file.

/*
Package websocket provides high-level bindings for the browser's WebSocket API.

These bindings offer a Dial function that returns a regular net.Conn.
It can be used similarly to net package.

	conn, err := websocket.Dial("ws://localhost/socket") // Blocks until connection is established.
	if err != nil {
		// handle error
	}

	buf := make([]byte, 1024)
	n, err = conn.Read(buf) // Blocks until a WebSocket frame is received.
	doSomethingWithData(buf[:n])
	if err != nil {
		// handle error
	}

	_, err = conn.Write([]byte("Hello!"))
	// ...

	err = conn.Close()
	// ...
*/
package websocket
