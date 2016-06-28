package httptesting

import (
	"net/http"
	"testing"
)

func checkStatus(t *testing.T, url string, status int) {
	if rsp, err := http.Get(url); err == nil {
		if rsp.StatusCode != status {
			t.Error("failed to use proper handler")
		}
	} else {
		t.Error(err)
	}
}

func TestCanOpenServersButALot(t *testing.T) {
	const total = 1 << 17
	const parallel = 1 << 3

	p := NewServerPool()
	defer p.Close()

	for i := 0; i < total; i++ {
		func() {
			for j := 0; j < parallel; j++ {
				s := p.Get(OK)
				defer p.Release(s)
			}
		}()
	}
}

func TestUsesNewHandler(t *testing.T) {
	p := NewServerPool()
	defer p.Close()

	s1 := p.Get(OK)
	checkStatus(t, s1.URL, http.StatusOK)
	p.Release(s1)

	s2 := p.Get(Teapot)
	checkStatus(t, s1.URL, http.StatusTeapot)
	p.Release(s2)
}

func TestReusesServer(t *testing.T) {
	p := NewServerPool()
	defer p.Close()

	s1 := p.Get(OK)
	p.Release(s1)
	s2 := p.Get(Teapot)

	if s1 != s2 {
		t.Error("failed to reuse server")
	}

	p.Release(s2)
}

func TestClosesIdle(t *testing.T) {

	p := NewServerPool()
	defer p.Close()

	s1 := p.Get(OK)
	s2 := p.Get(Teapot)
	s3 := p.Get(Teapot)

	p.Release(s1)
	p.Release(s2)

	checkStatus(t, s1.URL, http.StatusNotFound)
	checkStatus(t, s2.URL, http.StatusNotFound)
	checkStatus(t, s3.URL, http.StatusTeapot)

	p.Release(s3)
}
