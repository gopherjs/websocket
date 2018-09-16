package websocket_test

import (
	"flag"
	"fmt"
	"syscall/js"

	"github.com/LinearZoetrope/testevents"
)

func testStarted(ev testevents.Event) {
	document := js.Global().Get("document")
	body := document.Get("body")

	outsideElement := document.Call("createElement", "div")
	outsideElement.Set("id", ev.Name)

	nameElement := document.Call("createElement", "span")
	nameElement.Set("textContent", fmt.Sprintf("%s:", ev.Name))
	outsideElement.Call("appendChild", nameElement)

	nbspElement := document.Call("createElement", "span")
	nbspElement.Set("innerHTML", "&nbsp;")
	outsideElement.Call("appendChild", nbspElement)

	body.Call("appendChild", outsideElement)
}

func testPassed(ev testevents.Event) {
	document := js.Global().Get("document")
	outsideElement := document.Call("getElementById", ev.Name)

	statusElement := document.Call("createElement", "span")
	statusElement.Set("textContent", "PASSED")
	outsideElement.Call("appendChild", statusElement)
}

func testFailed(ev testevents.Event) {
	document := js.Global().Get("document")
	outsideElement := document.Call("getElementById", ev.Name)

	statusElement := document.Call("createElement", "span")
	statusElement.Set("textContent", "FAILED")
	outsideElement.Call("appendChild", statusElement)
}

func init() {
	testevents.Register(testevents.TestStarted, testevents.EventListener{testStarted, int(testevents.TestStarted)})
	testevents.Register(testevents.TestPassed, testevents.EventListener{testPassed, int(testevents.TestPassed)})
	testevents.Register(testevents.TestFailed, testevents.EventListener{testFailed, int(testevents.TestFailed)})

	flag.Set("test.v", "true")
	flag.Set("test.parallel", "4")
}

func getWSBaseURL() string {
	location := js.Global().Get("window").Get("document").Get("location")

	wsProtocol := "ws"
	if location.Get("protocol").String() == "https:" {
		wsProtocol = "wss"
	}

	return fmt.Sprintf("%s://%s:%s/ws/", wsProtocol, location.Get("hostname").String(), location.Get("port").String())
}
