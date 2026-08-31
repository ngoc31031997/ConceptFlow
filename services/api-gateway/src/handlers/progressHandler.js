'use strict';

/**
 * Factory managing the SSE side of the AMQP-to-SSE bridge (LLD Flow 3).
 *
 * Holds a `Map<projectId, Response[]>` connection registry in memory —
 * intentionally not persisted (logical-components.md): if the Gateway
 * restarts, SSE clients simply reconnect. Safe without locking because
 * Node's single-threaded event loop means no two handlers ever mutate the
 * map concurrently (dependency-injection.md "Concurrency Model").
 *
 * @param {{ start: () => void }} amqpClient - not used directly to receive
 *   messages (the composition root wires `onMessage` to `dispatch` below),
 *   but accepted for parity with the DI contract in dependency-injection.md
 *   and so callers/tests can pass a fake without the handler reaching into
 *   AMQP internals.
 * @returns {{
 *   subscribe: import('express').RequestHandler,
 *   dispatch: (message: { project_id: string, [key: string]: any }) => void,
 * }}
 */
function progressHandler(_amqpClient) {
  /** @type {Map<string, import('express').Response[]>} */
  const connections = new Map();

  /**
   * Express handler for `GET /v1/progress/:id` — registers the response as
   * an open SSE stream and cleans up when the client disconnects.
   */
  function subscribe(req, res) {
    const projectId = sanitizeProjectId(req.params.id);
    if (!projectId) {
      res.status(400).json({ error: 'invalid_project_id' });
      return;
    }

    res.status(200);
    res.set({
      'Content-Type': 'text/event-stream',
      'Cache-Control': 'no-cache',
      Connection: 'keep-alive',
    });
    if (typeof res.flushHeaders === 'function') res.flushHeaders();

    const existing = connections.get(projectId) || [];
    existing.push(res);
    connections.set(projectId, existing);

    req.on('close', () => {
      const remaining = (connections.get(projectId) || []).filter((r) => r !== res);
      if (remaining.length > 0) {
        connections.set(projectId, remaining);
      } else {
        connections.delete(projectId);
      }
    });
  }

  /**
   * Called by the AMQP consumer for every `progress.fanout` delivery.
   * If nobody is currently subscribed to this project, the message is
   * dropped silently — no replay buffer, at-most-once delivery is an
   * accepted trade-off since progress is UX-only (messaging-design.md).
   */
  function dispatch(message) {
    const projectId = message && message.project_id;
    if (!projectId) return;
    const subscribers = connections.get(projectId);
    if (!subscribers || subscribers.length === 0) return;

    const payload = `data: ${JSON.stringify(message)}\n\n`;
    for (const res of subscribers) {
      try {
        res.write(payload);
      } catch {
        // A broken browser connection is not fatal here — the AMQP message
        // is still acked by amqpClient regardless of this outcome
        // (messaging-design.md's explicit ack-regardless rule).
      }
    }
  }

  return { subscribe, dispatch };
}

/**
 * Minimal path-param sanitization only (NFR Design's explicit boundary:
 * deep body validation is downstream's job, not the Gateway's) — guards
 * against path traversal / control characters in a value used only as a
 * Map key here, never as a filesystem path.
 */
function sanitizeProjectId(raw) {
  if (typeof raw !== 'string') return null;
  if (!/^[A-Za-z0-9_-]+$/.test(raw)) return null;
  return raw;
}

module.exports = { progressHandler };
