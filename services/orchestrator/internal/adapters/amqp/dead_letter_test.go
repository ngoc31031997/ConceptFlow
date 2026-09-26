package amqp

import (
	"strings"
	"testing"

	amqp "github.com/rabbitmq/amqp091-go"
)

func TestDeadLetterMessageExplainsWhy(t *testing.T) {
	rejected := amqp.Delivery{Headers: amqp.Table{"x-death": []interface{}{amqp.Table{"reason": "rejected"}}}}
	if got := deadLetterMessage(rejected); !strings.Contains(got, "lỗi hạ tầng") {
		t.Errorf("rejected: got %q", got)
	}
	expired := amqp.Delivery{Headers: amqp.Table{"x-death": []interface{}{amqp.Table{"reason": "expired"}}}}
	if got := deadLetterMessage(expired); !strings.Contains(got, "24 giờ") {
		t.Errorf("expired: got %q", got)
	}
	if got := deadLetterMessage(amqp.Delivery{}); got == "" {
		t.Error("no header: want a generic message")
	}
}

func TestBumpEventAttemptCountsAndClears(t *testing.T) {
	const id = "test-msg"
	clearEventAttempts(id)
	for want := 1; want <= maxEventAttempts; want++ {
		if got := bumpEventAttempt(id); got != want {
			t.Fatalf("attempt %d: got %d", want, got)
		}
	}
	clearEventAttempts(id)
	if got := bumpEventAttempt(id); got != 1 {
		t.Errorf("after clear: got %d, want 1", got)
	}
	clearEventAttempts(id)
}
