// Copyright 2014-2015 GopherJS Team. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be found
// in the LICENSE file.

// This file has been added to make the exposed API of the internal js wrapper

package websocket

import "github.com/gopherjs/websocket/core"

// ReadyState is the top level export of gopherjs/websocket/core's object
type ReadyState core.ReadyState

// WebSocket is the top level export of gopherjs/websocket/core's object
type WebSocket core.WebSocket

// Various ReadyState top level export of gopherjs/websocket/core's object
const (
	Connecting ReadyState = ReadyState(core.Connecting)
	Open       ReadyState = ReadyState(core.Open)
	Closing    ReadyState = ReadyState(core.Closing)
	Closed     ReadyState = ReadyState(core.Closed)
)

// New is the top level export of gopherjs/websocket/core's object
var New = core.New
