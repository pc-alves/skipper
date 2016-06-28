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

var zeroHandler = http.HandlerFunc(func(rsp http.ResponseWriter, _ *http.Request) {
	rsp.WriteHeader(http.StatusNotFound)
})

var (
	Pool = NewServerPool()

	OK = http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {})

	Teapot = http.HandlerFunc(func(rsp http.ResponseWriter, _ *http.Request) {
		rsp.WriteHeader(http.StatusTeapot)
	})
)

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
	s[si].handler = zeroHandler
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
				m.response <- nil
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
	m := message{server: s, response: make(chan *httptest.Server)}
	sp.release <- m
	<-m.response
}

func (sp *ServerPool) Close() {
	close(sp.quit)
}

func WithServers(h []http.Handler, f func([]*httptest.Server)) {
	s := make([]*httptest.Server, len(h))
	for i, hi := range h {
		s[i] = Pool.Get(hi)
		defer Pool.Release(s[i])
	}

	f(s)
}

func WithServer(h http.Handler, f func(*httptest.Server)) {
	WithServers([]http.Handler{h}, func(s []*httptest.Server) {
		f(s[0])
	})
}
