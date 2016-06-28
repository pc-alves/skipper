package httptesting

import (
	"net/http"
	"net/http/httptest"
)

type handler struct {
	handler http.Handler
	busy    bool
}

type servers map[*httptest.Server]*handler

type message struct {
	handler  http.Handler
	server   *httptest.Server
	response chan *httptest.Server
}

type ServerPool struct {
	get, release chan message
	quit         chan struct{}
}

func (h *handler) ServeHTTP(rsp http.ResponseWriter, req *http.Request) {
	h.handler.ServeHTTP(rsp, req)
}

func (s servers) get(h http.Handler) *httptest.Server {
	for si, hi := range s {
		if !hi.busy {
			hi.handler = h
			hi.busy = true
			return si
		}
	}

	hi := &handler{h, true}
	si := httptest.NewServer(hi)
	s[si] = hi
	return si
}

func (s servers) release(si *httptest.Server) {
	s[si].handler = nil
	s[si].busy = false
}

func (s servers) closePool() {
	for si, hi := range s {
		if !hi.busy {
			si.Close()
		}
	}
}

func NewServerPool() *ServerPool {
	s := make(servers)
	get := make(chan message)
	release := make(chan message)
	quit := make(chan struct{})

	go func() {
		for {
			select {
			case m := <-get:
				m.response <- s.get(m.handler)
			case m := <-release:
				s.release(m.server)
			case <-quit:
				s.closePool()
				return
			}
		}
	}()

	return &ServerPool{get, release, quit}
}

func (sp *ServerPool) Get(h http.Handler) *httptest.Server {
	m := message{handler: h, response: make(chan *httptest.Server)}
	sp.get <- m
	return <-m.response
}

func (sp *ServerPool) Release(s *httptest.Server) {
	go func() { sp.release <- message{server: s} }()
}

func (sp *ServerPool) Close() {
	close(sp.quit)
}
