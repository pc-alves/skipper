package httptesting

import (
	"net/http"
	"testing"
)

func TestCanOpenServersButALot(t *testing.T) {
	const total = 1 << 17
	const parallel = 1 << 3

	p := NewServerPool()
	ok := http.HandlerFunc(func(rsp http.ResponseWriter, _ *http.Request) {})

	for i := 0; i < total; i++ {
		func() {
			for j := 0; j < parallel; j++ {
				s := p.Get(ok)
				defer p.Release(s)
			}
		}()
	}
}
