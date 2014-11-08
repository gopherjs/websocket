package websocket

import (
	"bytes"
	"errors"
	"io"
	"net/url"
	"runtime"
	"time"

	"github.com/gopherjs/gopherjs/js"
	"honnef.co/go/js/dom"
	"honnef.co/go/js/util"
)

type ReadyState uint16

const (
	// Ready state constants from
	// https://developer.mozilla.org/en-US/docs/Web/API/WebSocket#Ready_state_constants
	Connecting ReadyState = 0 // The connection is not yet open.
	Open                  = 1 // The connection is open and ready to communicate.
	Closing               = 2 // The connection is in the process of closing.
	Closed                = 3 // The connection is closed or couldn't be opened.
)

var (
	ErrSocketClosed = errors.New("the socket has been closed")
)

type Addr struct {
	*url.URL
}

// Network returns the network type for a WebSocket, "websocket".
func (addr *Addr) Network() string { return "websocket" }

type receiveItem struct {
	Error error
	Event *dom.MessageEvent
}

type WebSocket struct {
	js.Object
	util.EventTarget

	// See https://developer.mozilla.org/en-US/docs/Web/API/WebSocket#Attributes
	// for information about these attributes.
	BinaryType     string     `js:"binaryType"`
	BufferedAmount uint32     `js:"bufferedAmount"`
	Extensions     string     `js:"extensions"`
	Protocol       string     `js:"protocol"`
	ReadyState     ReadyState `js:"readyState"`
	URL            string     `js:"url"`

	openCh chan *js.Error

	ch      chan *receiveItem
	readBuf *bytes.Reader
}

// New creates a new WebSocket. It blocks until the connection opens or throws
// an error.
func New(url string) (*WebSocket, error) {
	object := js.Global.Get("WebSocket").New(url)

	ws := &WebSocket{
		Object:      object,
		EventTarget: util.EventTarget{Object: object},
		ch:          make(chan *receiveItem),
		openCh:      make(chan *js.Error),
	}
	ws.init()

	// Wait for the WebSocket to open or error. See: onOpen & onClose.
	err, ok := <-ws.openCh
	if ok && err != nil {
		ws.Close() // Just in case the connection was open for some reason?
		return nil, err
	}

	return ws, nil
}

func (ws *WebSocket) init() {
	// Some browsers don't support Blobs. On top of that, []byte is converted to
	// Int8Array, which is handled similarly to ArrayBuffer.
	ws.BinaryType = "arraybuffer"

	// Add all of the event handlers.
	ws.EventTarget.AddEventListener("open", false, ws.onOpen)
	ws.EventTarget.AddEventListener("close", false, ws.onClose)
	ws.EventTarget.AddEventListener("error", false, ws.onError)
	ws.EventTarget.AddEventListener("message", false, ws.onMessage)
}

func (ws *WebSocket) onOpen(event js.Object) {
	close(ws.openCh)
}

func (ws *WebSocket) onClose(event js.Object) {
	if wasClean := event.Get("wasClean").Bool(); !wasClean {
		go func() {
			defer func() {
				// This feels extremely hacky, but I can't think of a better way
				// to do it. openCh is closed before the end of New(), but this
				// is one of the paths that can close it. The other is in
				// WebSocket.onOpen.
				e := recover()
				if e == nil {
					return
				}
				if e, ok := e.(runtime.Error); ok && e.Error() == "runtime error: send on closed channel" {
					return
				}
				panic(e)
			}()

			// If the close wasn't clean, we need to inform the openCh. This
			// allows New to return an error.
			ws.openCh <- &js.Error{Object: event}
			close(ws.openCh)
		}()
	}
	close(ws.ch)
}

func (ws *WebSocket) onError(event js.Object) {
	// TODO: Don't send to ws.ch when this is a connection error.
	// onError is called when a connection fails. Such errors shouldn't be sent
	// to ws.ch.
	go func() {
		// This allows Receive to return an error. It seems that many
		// WebSocket.send errors are handled this way.
		ws.ch <- &receiveItem{
			Event: nil,
			Error: &js.Error{Object: event},
		}
	}()
}

func (ws *WebSocket) onMessage(event js.Object) {
	go func() {
		ws.ch <- &receiveItem{
			Event: dom.WrapEvent(event).(*dom.MessageEvent),
			Error: nil,
		}
	}()
}

// sendRaw sends a message on the WebSocket. The data argument can be a string
// or a js.Object containing an ArrayBuffer.
//
// The helper functions WriteString and SendBinary should be preferred to this.
func (ws *WebSocket) sendRaw(data interface{}) (err error) {
	defer func() {
		e := recover()
		if e == nil {
			return
		}
		if jsErr, ok := e.(*js.Error); ok && jsErr != nil {
			err = jsErr
		} else {
			panic(e)
		}
	}()
	ws.Object.Call("send", data)
	return nil
}

// ReceiveFrame receives one full frame from the WebSocket. It blocks until the
// frame is received.
func (ws *WebSocket) ReceiveFrame() (*dom.MessageEvent, error) {
	item, ok := <-ws.ch
	if !ok { // The channel has been closed
		return nil, ErrSocketClosed
	}
	return item.Event, item.Error
}

// SendString sends a string on the WebSocket. This is a helper method that
// calls SendRaw.
func (ws *WebSocket) WriteString(data string) (n int, err error) {
	err = ws.sendRaw(data)
	if err != nil {
		n = len(data)
	}
	return
}

// Read implements the io.Reader interface:
// It reads the data of a frame from the WebSocket connection. If b is not large
// enough, the next Read will read the rest of that frame.
func (ws *WebSocket) Read(b []byte) (n int, err error) {
	// TODO: Make this fully full the buffer once the Timeout functions are
	// implemented.

	if ws.readBuf != nil {
		n, err = ws.readBuf.Read(b)
		if err == io.EOF {
			ws.readBuf = nil
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

	frame, err := ws.ReceiveFrame()
	if err != nil {
		return 0, err
	}

	receivedBytes := []byte(frame.Data.Str())
	// fast path
	if len(b) >= len(receivedBytes) {
		n = copy(b, receivedBytes)
		return
	}

	ws.readBuf = bytes.NewReader(receivedBytes)

	n, err = ws.readBuf.Read(b)
	if err == io.EOF {
		ws.readBuf = nil
		err = nil
	}
	return
}

// Write sends binary data on the WebSocket.
//
// Note: There are cases where the browser will throw an exception if it
// believes that the data passed to this function may be UTF-8.
func (ws *WebSocket) Write(p []byte) (n int, err error) {
	// We use Write to conform with the io.Writer interface.
	err = ws.sendRaw(p)
	if err != nil {
		n = len(p)
	}
	return
}

// Close closes the underlying WebSocket and cleans up any resources associated
// with the helper.
func (ws *WebSocket) Close() (err error) {
	defer func() {
		e := recover()
		if e == nil {
			return
		}
		if jsErr, ok := e.(*js.Error); ok && jsErr != nil {
			err = jsErr
		} else {
			panic(e)
		}
	}()
	ws.Object.Call("close")
	return nil
}

func (ws *WebSocket) LocalAddr() *Addr {
	// We can't use net.Addr, because net.init() causes a panic due to attempts
	// to make syscalls.

	// TODO: Find a more graceful way to handle this.
	panic("It is not possible to implement this function due to limitations within JavaScript")
}

func (ws *WebSocket) RemoteAddr() *Addr {
	// We can't use net.Addr, because net.init() causes a panic due to attempts
	// to make syscalls.

	wsURL, err := url.Parse(ws.URL)
	if err != nil {
		// TODO: Should we be panicking for this?
		panic(err)
	}
	return &Addr{wsURL}
}

func (ws *WebSocket) SetDeadline(t time.Time) error {
	// TODO: Implement
	panic("not yet implemeneted")
}

func (ws *WebSocket) SetReadDeadline(t time.Time) error {
	// TODO: Implement
	panic("not yet implemeneted")
}

func (ws *WebSocket) SetWriteDeadline(t time.Time) error {
	// TODO: Implement
	panic("not yet implemeneted")
}
