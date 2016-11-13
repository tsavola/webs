package webs

import (
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

// Conn
type Conn struct {
	ws *websocket.Conn
}

func (c *Conn) SetEvalDeadline(t time.Time) error {
	return c.ws.SetWriteDeadline(t)
}

func (c *Conn) Eval(stmt string) error {
	return c.ws.WriteMessage(websocket.TextMessage, []byte(stmt))
}

func (c *Conn) ReadMessage() (data []byte, err error) {
	_, data, err = c.ws.ReadMessage()
	return
}

// Handler
type Handler interface {
	ServeConn(*Conn)
}

// HandlerFunc
type HandlerFunc func(*Conn)

func (f HandlerFunc) ServeConn(c *Conn) {
	f(c)
}

// Init
func Init(mux *http.ServeMux, path string, h Handler) {
	if mux == nil {
		mux = http.DefaultServeMux
	}

	mux.HandleFunc(path, handleIndex)
	mux.Handle(path+"io", ioHandler{h})
}

func handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(indexHTML)
}

//
type ioHandler struct {
	connHandler Handler
}

func (io ioHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	u := websocket.Upgrader{
		CheckOrigin: func(_ *http.Request) bool {
			return true
		},
	}

	c, err := u.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer c.Close()

	io.connHandler.ServeConn(&Conn{c})
}
