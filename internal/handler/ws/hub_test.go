package ws

import (
	"testing"
)

func TestHub_BroadcastReadExcludesReader(t *testing.T) {
	t.Parallel()

	hub := NewHub()
	reader := newClient(hub, nil, 1)
	other := newClient(hub, nil, 2)
	hub.Register(reader)
	hub.Register(other)

	payload := []byte(`{"type":"read","chat_id":10,"user_id":1,"last_read_message_id":5}`)
	hub.BroadcastRead(10, 1, payload, []int64{1, 2})

	select {
	case got := <-other.send:
		if string(got) != string(payload) {
			t.Fatalf("payload = %s, want %s", got, payload)
		}
	default:
		t.Fatal("expected delivery to other participant")
	}

	select {
	case <-reader.send:
		t.Fatal("read event must not be delivered to the reader")
	default:
	}
}
