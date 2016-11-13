package webs_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"testing"
	"time"

	"."
	"./dom"
)

const (
	ioTimeout = time.Second
)

type tester struct {
	doc *dom.Document

	countMutex sync.Mutex
	countModel int
	countView  *dom.Element
}

func Test(t *testing.T) {
	tester := new(tester)

	tester.doc = dom.NewDocument(testStyle)
	tester.doc.SetTitle("test")

	p := tester.doc.NewElement("p")
	tester.doc.Body.Append(p)

	button := tester.doc.NewElement("button")
	button.Set("innerText", "+1")
	button.SetFunction("onclick", `webs.send(JSON.stringify({Act: "increment-count"}))`)
	p.Append(button)

	tester.countView = tester.doc.NewElement("span")
	tester.countView.Set("innerText", strconv.Itoa(tester.countModel))
	tester.countView.Set("className", "count")
	p.Append(tester.countView)

	go func() {
		div := tester.doc.NewElement("div")
		p.Append(div)

		for t := range time.NewTicker(time.Second).C {
			div.Set("innerText", t.String())
		}
	}()

	mux := http.NewServeMux()
	webs.Init(mux, "/", tester)

	if err := http.ListenAndServe("localhost:8080", mux); err != nil {
		fmt.Println(err)
		return
	}
}

func (tester *tester) ServeConn(c *webs.Conn) {
	go func() {
		for {
			data, err := c.ReadMessage()
			if err != nil {
				fmt.Println(err)
				break
			}

			var msg struct {
				Act string
			}

			if err := json.Unmarshal(data, &msg); err != nil {
				panic(err)
			}

			switch msg.Act {
			case "increment-count":
				tester.countMutex.Lock()
				tester.countModel++
				tester.countView.Set("innerText", strconv.Itoa(tester.countModel))
				tester.countMutex.Unlock()

			default:
				panic(msg.Act)
			}
		}
	}()

	stmts, unsubscribe := tester.doc.Subscribe()
	defer unsubscribe()

	for s := range stmts {
		if err := c.SetEvalDeadline(time.Now().Add(ioTimeout)); err != nil {
			panic(err)
		}

		if err := c.Eval(s); err != nil {
			fmt.Println(err)
			break
		}
	}
}
