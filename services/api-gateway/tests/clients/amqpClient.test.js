'use strict';

const { createAmqpClient } = require('../../src/clients/amqpClient');

function makeFakeChannel() {
  return {
    assertExchange: jest.fn().mockResolvedValue(undefined),
    assertQueue: jest.fn().mockResolvedValue({ queue: 'fake-exclusive-queue' }),
    bindQueue: jest.fn().mockResolvedValue(undefined),
    consume: jest.fn(),
    ack: jest.fn(),
    close: jest.fn().mockResolvedValue(undefined),
  };
}

function makeFakeConnection(channel) {
  const handlers = {};
  return {
    createChannel: jest.fn().mockResolvedValue(channel),
    on: jest.fn((event, cb) => {
      handlers[event] = cb;
    }),
    close: jest.fn().mockResolvedValue(undefined),
    _handlers: handlers,
  };
}

const silentLogger = { info: () => {}, error: () => {}, warn: () => {} };

describe('amqpClient', () => {
  test('connects, declares exclusive queue bound to progress.fanout, consumes and acks', async () => {
    const channel = makeFakeChannel();
    const connection = makeFakeConnection(channel);
    const amqplibImpl = { connect: jest.fn().mockResolvedValue(connection) };
    const onMessage = jest.fn();

    const client = createAmqpClient('amqp://test', onMessage, { amqplibImpl, logger: silentLogger });
    client.start();
    await flushPromises();

    expect(amqplibImpl.connect).toHaveBeenCalledWith('amqp://test');
    expect(channel.assertExchange).toHaveBeenCalledWith('progress.fanout', 'fanout', { durable: true });
    expect(channel.assertQueue).toHaveBeenCalledWith('', { exclusive: true });
    expect(channel.bindQueue).toHaveBeenCalledWith('fake-exclusive-queue', 'progress.fanout', '');

    const consumeCallback = channel.consume.mock.calls[0][1];
    const fakeMsg = { content: Buffer.from(JSON.stringify({ project_id: 'p1', step: 'render', status: 'in_progress' })) };
    consumeCallback(fakeMsg);

    expect(onMessage).toHaveBeenCalledWith({ project_id: 'p1', step: 'render', status: 'in_progress' });
    expect(channel.ack).toHaveBeenCalledWith(fakeMsg);
  });

  test('acks message even if onMessage callback throws', async () => {
    const channel = makeFakeChannel();
    const connection = makeFakeConnection(channel);
    const amqplibImpl = { connect: jest.fn().mockResolvedValue(connection) };
    const onMessage = jest.fn(() => {
      throw new Error('SSE write failed');
    });

    const client = createAmqpClient('amqp://test', onMessage, { amqplibImpl, logger: silentLogger });
    client.start();
    await flushPromises();

    const consumeCallback = channel.consume.mock.calls[0][1];
    const fakeMsg = { content: Buffer.from(JSON.stringify({ project_id: 'p1' })) };
    consumeCallback(fakeMsg);

    expect(channel.ack).toHaveBeenCalledWith(fakeMsg);
  });

  test('reconnects with fixed 5s delay when connect() fails', async () => {
    jest.useFakeTimers();
    const channel = makeFakeChannel();
    const connection = makeFakeConnection(channel);
    const amqplibImpl = {
      connect: jest.fn().mockRejectedValueOnce(new Error('ECONNREFUSED')).mockResolvedValueOnce(connection),
    };

    const client = createAmqpClient('amqp://test', jest.fn(), {
      amqplibImpl,
      logger: silentLogger,
      reconnectDelayMs: 5000,
    });
    client.start();
    await flushPromises();

    expect(amqplibImpl.connect).toHaveBeenCalledTimes(1);

    jest.advanceTimersByTime(5000);
    await flushPromises();

    expect(amqplibImpl.connect).toHaveBeenCalledTimes(2);
    jest.useRealTimers();
  });

  test('reconnects when connection emits close event', async () => {
    jest.useFakeTimers();
    const channel = makeFakeChannel();
    const connection = makeFakeConnection(channel);
    const amqplibImpl = { connect: jest.fn().mockResolvedValue(connection) };

    const client = createAmqpClient('amqp://test', jest.fn(), {
      amqplibImpl,
      logger: silentLogger,
      reconnectDelayMs: 5000,
    });
    client.start();
    await flushPromises();

    expect(amqplibImpl.connect).toHaveBeenCalledTimes(1);
    connection._handlers.close();

    jest.advanceTimersByTime(5000);
    await flushPromises();

    expect(amqplibImpl.connect).toHaveBeenCalledTimes(2);
    jest.useRealTimers();
  });
});

async function flushPromises() {
  for (let i = 0; i < 5; i += 1) {
    await Promise.resolve();
  }
}
