package socket

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestNotifierBroadcastWakeup(t *testing.T) {
	n := NewNotifier()
	go n.Poll()

	srv := httptest.NewServer(http.HandlerFunc(n.Handle))
	t.Cleanup(srv.Close)

	url := "ws" + strings.TrimPrefix(srv.URL, "http") + "/"
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		n.Broadcast()
		_ = conn.SetReadDeadline(time.Now().Add(150 * time.Millisecond))
		_, msg, err := conn.ReadMessage()
		if err != nil {
			continue
		}
		if string(msg) != "wakeup" {
			t.Fatalf("got %q want wakeup", msg)
		}
		return
	}
	t.Fatal("timed out waiting for wakeup")
}
