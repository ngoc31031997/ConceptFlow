'use strict';

const amqplib = require('amqplib');

const RECONNECT_DELAY_MS = 5_000;
const EXCHANGE_NAME = 'progress.fanout';

/**
 * Creates a RabbitMQ consumer bound to the `progress.fanout` exchange via
 * an exclusive queue owned by this Gateway instance (messaging-design.md).
 *
 * Reconnects on a fixed 5s delay with no retry limit — the connection is
 * to an internal Docker-network broker, so there is never a reason to
 * "give up" permanently (nfr-design-patterns.md's Resilience Pattern).
 * A dropped AMQP connection does not affect already-open SSE connections;
 * it just means progress updates stop arriving until reconnect succeeds
 * (NFR Requirements' Availability section).
 *
 * @param {string} rabbitmqUrl
 * @param {(message: object) => void} onMessage - called with the parsed
 *   ProgressMessage payload for every delivery.
 * @param {{ amqplibImpl?: typeof amqplib, reconnectDelayMs?: number, logger?: Console }} [options]
 * @returns {{ start: () => void, stop: () => Promise<void> }}
 */
function createAmqpClient(rabbitmqUrl, onMessage, options = {}) {
  const amqp = options.amqplibImpl || amqplib;
  const reconnectDelayMs = options.reconnectDelayMs ?? RECONNECT_DELAY_MS;
  const logger = options.logger || console;

  let connection = null;
  let channel = null;
  let stopped = false;
  let reconnectTimer = null;

  async function connectAndConsume() {
    if (stopped) return;
    try {
      connection = await amqp.connect(rabbitmqUrl);
      connection.on('error', (err) => {
        logger.error({ err }, 'AMQP connection error');
      });
      connection.on('close', () => {
        if (!stopped) scheduleReconnect();
      });

      channel = await connection.createChannel();
      await channel.assertExchange(EXCHANGE_NAME, 'fanout', { durable: true });
      const { queue } = await channel.assertQueue('', { exclusive: true });
      await channel.bindQueue(queue, EXCHANGE_NAME, '');

      channel.consume(queue, (msg) => {
        if (!msg) return;
        try {
          const payload = JSON.parse(msg.content.toString('utf8'));
          onMessage(payload);
        } catch (err) {
          logger.error({ err }, 'Failed to parse/handle progress message');
        } finally {
          // Ack regardless of fan-out/SSE-write outcome (messaging-design.md):
          // a broken browser connection is not a reason to requeue an AMQP message.
          channel.ack(msg);
        }
      });

      logger.info({ queue }, 'AMQP consumer connected and bound to progress.fanout');
    } catch (err) {
      logger.error({ err }, 'Failed to connect to RabbitMQ, will retry');
      scheduleReconnect();
    }
  }

  function scheduleReconnect() {
    if (stopped || reconnectTimer) return;
    reconnectTimer = setTimeout(() => {
      reconnectTimer = null;
      connectAndConsume();
    }, reconnectDelayMs);
  }

  function start() {
    stopped = false;
    connectAndConsume();
  }

  async function stop() {
    stopped = true;
    if (reconnectTimer) {
      clearTimeout(reconnectTimer);
      reconnectTimer = null;
    }
    try {
      if (channel) await channel.close();
    } catch {
      // ignore
    }
    try {
      if (connection) await connection.close();
    } catch {
      // ignore
    }
  }

  return { start, stop };
}

module.exports = { createAmqpClient };
