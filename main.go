package websocket

import (
	"errors"
	"runtime"

	"github.com/gopherjs/gopherjs/js"
	"honnef.co/go/js/dom"
	"honnef.co/go/js/util"
)

// ReadyState represents the state that a WebSocket is in. For more information
// about the available states, see
// https://developer.mozilla.org/en-US/docs/Web/API/WebSocket#Ready_state_constants
type ReadyState uint16

func (rs ReadyState) String() string {
	switch rs {
	case Connecting:
		return "Connecting"
	case Open:
		return "Open"
	case Closing:
		return "Closing"
	case Closed:
		return "Closed"
	default:
		return "Unknown"
	}
}

// Ready state constants from
// https://developer.mozilla.org/en-US/docs/Web/API/WebSocket#Ready_state_constants
const (
	Connecting ReadyState = 0 // The connection is not yet open.
	Open                  = 1 // The connection is open and ready to communicate.
	Closing               = 2 // The connection is in the process of closing.
	Closed                = 3 // The connection is closed or couldn't be opened.
)

// ErrSocketClosed is returned when an operation is attempted on a
// closed socket.
var ErrSocketClosed = errors.New("the socket has been closed")

type receiveItem struct {
	Error error
	Event *dom.MessageEvent
}

// WebSocket is a convenience wrapper around the browser's WebSocket
// implementation. For more information, see
// https://developer.mozilla.org/en-US/docs/Web/API/WebSocket
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

	ch     chan *receiveItem
	openCh chan *js.Error
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

// SendRaw sends a message on the WebSocket. The data argument can be a string
// or a js.Object containing an ArrayBuffer.
//
// The helper functions SendString and SendBinary should be preferred to this.
func (ws *WebSocket) SendRaw(data interface{}) (err error) {
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

// SendString sends a string on the WebSocket. This is a helper method that
// calls SendRaw.
func (ws *WebSocket) SendString(data string) error {
	return ws.SendRaw(data)
}

// Write sends binary data on the WebSocket.
//
// Note: There are cases where the browser will throw an exception if it
// believes that the data passed to this function may be UTF-8.
func (ws *WebSocket) Write(p []byte) (int, error) {
	// We use Write to conform with the io.Writer interface.
	err := ws.SendRaw(p)
	if err != nil {
		return 0, err
	}
	return len(p), nil
}

// Receive receives one message from the WebSocket. It blocks until the message
// is received.
func (ws *WebSocket) Receive() (*dom.MessageEvent, error) {
	item, ok := <-ws.ch
	if !ok { // The channel has been closed
		return nil, ErrSocketClosed
	}
	return item.Event, item.Error
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
