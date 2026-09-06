package amqp

import (
	"context"
	"log/slog"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

// ConnectionManager owns the amqp091-go Connection/Channel lifecycle and is
// the single source of truth for "the current live channel". Publisher and
// Consumer never hold a *amqp.Channel directly — they ask ConnectionManager
// for it on every use, and register an OnReconnect callback if they need to
// redo per-channel setup (Consumer's queue subscriptions) after a reconnect.
//
// This exists because a bare amqp091-go channel does not recover on its
// own: once the broker closes it (restart, network blip), every further
// call fails forever with "channel/connection is not open". Before this
// existed, Orchestrator's outbox relay looped on that dead channel every
// OUTBOX_POLL_INTERVAL_MS with no way to recover, permanently stalling
// command dispatch (see ADR-0022).
type ConnectionManager struct {
	url          string
	initialDelay time.Duration
	maxDelay     time.Duration
	logger       *slog.Logger

	mu      sync.RWMutex
	conn    *amqp.Connection
	channel *amqp.Channel

	onReconnectMu sync.Mutex
	onReconnect   []func(ch *amqp.Channel) error
}

// NewConnectionManager constructs a ConnectionManager. initialDelay/maxDelay
// bound the exponential backoff used while reconnecting.
func NewConnectionManager(url string, initialDelay, maxDelay time.Duration, logger *slog.Logger) *ConnectionManager {
	if logger == nil {
		logger = slog.Default()
	}
	return &ConnectionManager{url: url, initialDelay: initialDelay, maxDelay: maxDelay, logger: logger}
}

// Connect dials the broker and opens the first channel, failing fast if
// RabbitMQ is unreachable at startup (same behavior as the direct
// amqplib.Dial/conn.Channel calls this replaces in main.go). Once connected,
// it starts the background watchdog that reconnects on close.
func (m *ConnectionManager) Connect(ctx context.Context) error {
	conn, channel, err := dial(m.url)
	if err != nil {
		return err
	}
	m.mu.Lock()
	m.conn = conn
	m.channel = channel
	m.mu.Unlock()

	go m.watch(ctx, conn, channel)
	return nil
}

// Channel returns the current live channel. Callers must call this again
// after any error rather than caching the result — the returned pointer can
// change underneath them at any time (reconnect).
func (m *ConnectionManager) Channel() *amqp.Channel {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.channel
}

// OnReconnect registers fn to run (with the new channel) each time a
// reconnect succeeds. Used by Consumer to re-issue its queue subscriptions
// — its previous delivery loops already exited on their own when the old
// channel closed (channel.Consume's returned Go channel closes with it), so
// there is nothing to tear down first.
func (m *ConnectionManager) OnReconnect(fn func(ch *amqp.Channel) error) {
	m.onReconnectMu.Lock()
	defer m.onReconnectMu.Unlock()
	m.onReconnect = append(m.onReconnect, fn)
}

// watch waits for the connection or channel to close, then reconnects with
// capped exponential backoff, notifying registered OnReconnect callbacks on
// success. It loops for the lifetime of ctx.
func (m *ConnectionManager) watch(ctx context.Context, conn *amqp.Connection, channel *amqp.Channel) {
	connClosed := conn.NotifyClose(make(chan *amqp.Error, 1))
	chanClosed := channel.NotifyClose(make(chan *amqp.Error, 1))

	select {
	case <-ctx.Done():
		return
	case err := <-connClosed:
		m.logger.WarnContext(ctx, "amqp connection closed, reconnecting", "error", err)
	case err := <-chanClosed:
		m.logger.WarnContext(ctx, "amqp channel closed, reconnecting", "error", err)
	}

	newConn, newChannel := m.reconnect(ctx)
	if newConn == nil {
		return // ctx cancelled while reconnecting
	}

	m.mu.Lock()
	m.conn = newConn
	m.channel = newChannel
	m.mu.Unlock()

	m.onReconnectMu.Lock()
	callbacks := append([]func(ch *amqp.Channel) error(nil), m.onReconnect...)
	m.onReconnectMu.Unlock()
	for _, fn := range callbacks {
		if err := fn(newChannel); err != nil {
			m.logger.ErrorContext(ctx, "amqp reconnect callback failed", "error", err)
		}
	}

	go m.watch(ctx, newConn, newChannel)
}

// reconnect redials with capped exponential backoff until it succeeds or
// ctx is cancelled (returning nil, nil in the latter case).
func (m *ConnectionManager) reconnect(ctx context.Context) (*amqp.Connection, *amqp.Channel) {
	delay := m.initialDelay
	for {
		select {
		case <-ctx.Done():
			return nil, nil
		case <-time.After(delay):
		}

		conn, channel, err := dial(m.url)
		if err == nil {
			m.logger.InfoContext(ctx, "amqp reconnected")
			return conn, channel
		}
		m.logger.ErrorContext(ctx, "amqp reconnect attempt failed, retrying", "error", err, "next_delay", delay)
		delay = nextBackoff(delay, m.maxDelay)
	}
}

// nextBackoff doubles delay, capped at max. Extracted for unit testing
// without a real broker.
func nextBackoff(delay, max time.Duration) time.Duration {
	delay *= 2
	if delay > max {
		delay = max
	}
	return delay
}

func dial(url string) (*amqp.Connection, *amqp.Channel, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, nil, err
	}
	channel, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, nil, err
	}
	return conn, channel, nil
}
