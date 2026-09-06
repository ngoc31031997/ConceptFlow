package amqp

import (
	"testing"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

func TestNextBackoff_DoublesUntilCap(t *testing.T) {
	max := 30 * time.Second
	cases := []struct {
		in   time.Duration
		want time.Duration
	}{
		{1 * time.Second, 2 * time.Second},
		{2 * time.Second, 4 * time.Second},
		{16 * time.Second, 30 * time.Second}, // would be 32s, capped at max
		{30 * time.Second, 30 * time.Second}, // already at cap, stays capped
	}
	for _, c := range cases {
		if got := nextBackoff(c.in, max); got != c.want {
			t.Errorf("nextBackoff(%v, %v) = %v, want %v", c.in, max, got, c.want)
		}
	}
}

func TestConnectionManager_OnReconnect_RegistersMultipleCallbacks(t *testing.T) {
	m := NewConnectionManager("amqp://unused", time.Second, 30*time.Second, nil)

	var calls []int
	m.OnReconnect(func(ch *amqp.Channel) error { calls = append(calls, 1); return nil })
	m.OnReconnect(func(ch *amqp.Channel) error { calls = append(calls, 2); return nil })

	m.onReconnectMu.Lock()
	callbacks := append([]func(ch *amqp.Channel) error(nil), m.onReconnect...)
	m.onReconnectMu.Unlock()

	for _, fn := range callbacks {
		_ = fn(nil)
	}
	if len(calls) != 2 || calls[0] != 1 || calls[1] != 2 {
		t.Fatalf("expected both callbacks to run in registration order, got %v", calls)
	}
}
