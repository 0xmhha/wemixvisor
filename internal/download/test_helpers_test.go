package download

import (
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
)

func newTestServer(t testing.TB, handler http.Handler) *httptest.Server {
	t.Helper()
	server := httptest.NewUnstartedServer(handler)
	ln, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Skipf("unable to create test listener: %v", err)
	}
	server.Listener = ln
	server.Start()
	return server
}
