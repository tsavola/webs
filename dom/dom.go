package dom

import (
	"encoding/json"
	"strconv"
	"sync/atomic"
)

var (
	idSequence uint64 // atomic
)

func marshal(x interface{}) string {
	b, err := json.Marshal(x)
	if err != nil {
		panic(err)
	}
	return string(b)
}

type request struct {
	subscriber chan<- string
	subscribe  bool
}

// Document
type Document struct {
	mods        chan func() string
	requests    chan request
	subscribers map[chan<- string]struct{}

	initStmt  string
	titleStmt string

	Body *Element
}

func NewDocument(style string) (doc *Document) {
	doc = &Document{
		mods:        make(chan func() string),
		requests:    make(chan request),
		subscribers: make(map[chan<- string]struct{}),

		initStmt: `document.getElementsByTagName("html")[0].innerHTML = "<head><style></style></head><body></body>";
			document.head.getElementsByTagName("style")[0].innerHTML = ` + marshal(style) + `;`,
	}

	doc.Body = &Element{
		doc: doc,
		tag: "body",
	}

	go doc.serve()

	return
}

func (doc *Document) serve() {
	for {
		select {
		case mod := <-doc.mods:
			if stmt := mod(); stmt != "" {
				for subscriber := range doc.subscribers {
					subscriber <- stmt
				}
			}

		case req := <-doc.requests:
			if req.subscribe {
				doc.subscribers[req.subscriber] = struct{}{}
				req.subscriber <- doc.recreateStmt()
			} else {
				delete(doc.subscribers, req.subscriber)
				close(req.subscriber)
			}
		}
	}
}

func (doc *Document) Subscribe() (stmts <-chan string, unsubscribe func()) {
	sub := make(chan string)
	doc.requests <- request{sub, true}

	unsubs := func() {
		reqs := doc.requests

		for reqs != nil && sub != nil {
			select {
			case reqs <- request{sub, false}:
				reqs = nil

			case _, ok := <-sub:
				if !ok {
					sub = nil
				}
			}
		}
	}

	return sub, unsubs
}

func (doc *Document) recreateStmt() string {
	return doc.initStmt + doc.titleStmt + doc.Body.recreateChildrenStmt()
}

func (doc *Document) SetTitle(title string) {
	stmt := `document.title =` + marshal(title) + `;`
	doc.mods <- func() string {
		doc.titleStmt = stmt
		return stmt
	}
}

// Element
type Element struct {
	doc *Document
	tag string
	id  string

	parent   *Element
	children []*Element

	props map[string]string
}

func (doc *Document) NewElement(tag string) *Element {
	return &Element{
		doc: doc,
		tag: tag,
		id:  strconv.FormatUint(atomic.AddUint64(&idSequence, 1), 16),
	}
}

func (e *Element) createExpr() (expr string) {
	expr = `(function() {
		var e = document.createElement("` + e.tag + `");
		e.id = "` + e.id + `";`
	for _, suffix := range e.props {
		expr += `e` + suffix
	}
	expr += `
		return e;
	})()`
	return
}

func (e *Element) getExpr() string {
	if e.id == "" {
		return `document.body`
	} else {
		return `document.getElementById("` + e.id + `")`
	}
}

func (e *Element) Append(child *Element) {
	e.doc.mods <- func() string {
		e.children = append(e.children, child)
		child.parent = e
		return e.getExpr() + `.appendChild(` + child.createExpr() + `);`
	}
}

func (e *Element) Remove() {
	e.doc.mods <- func() string {
		for i, sibling := range e.parent.children {
			if sibling == e {
				e.parent.children = append(e.parent.children[:i], e.parent.children[i+1:]...)
				e.parent = nil
				return e.getExpr() + `.remove();`
			}
		}
		panic("element not found in its parent")
	}
}

func (e *Element) recreateChildrenStmt() (stmt string) {
	stmt = `(function() {
		var e =` + e.getExpr() + `;`
	for _, child := range e.children {
		stmt += child.recreateStmt()
	}
	stmt += `})();`
	return
}

func (e *Element) recreateStmt() (stmt string) {
	stmt = `(function(parent) {
		var e =` + e.createExpr() + `;
		parent.appendChild(e);`
	for _, child := range e.children {
		stmt += child.recreateStmt()
	}
	stmt += `})(e);`
	return
}

func (e *Element) Set(name string, value interface{}) {
	e.set(name, marshal(value))
}

func (e *Element) SetFunction(name string, jsCode string) {
	e.set(name, `function() { `+jsCode+`}`)
}

func (e *Element) set(name, jsValue string) {
	suffix := `.` + name + `=` + jsValue + `;`
	e.doc.mods <- func() (stmt string) {
		if e.props == nil {
			e.props = make(map[string]string)
		}
		e.props[name] = suffix
		if e.parent != nil {
			stmt = e.getExpr() + suffix
		}
		return
	}
}
