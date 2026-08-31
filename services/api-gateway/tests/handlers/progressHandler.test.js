'use strict';

const { progressHandler } = require('../../src/handlers/progressHandler');

function makeFakeRes() {
  const listeners = {};
  return {
    status: jest.fn().mockReturnThis(),
    set: jest.fn().mockReturnThis(),
    json: jest.fn().mockReturnThis(),
    flushHeaders: jest.fn(),
    write: jest.fn(),
    _listeners: listeners,
  };
}

function makeFakeReq(params) {
  const handlers = {};
  return {
    params,
    on: jest.fn((event, cb) => {
      handlers[event] = cb;
    }),
    _trigger: (event) => handlers[event] && handlers[event](),
  };
}

describe('progressHandler', () => {
  test('subscribe registers connection and sets SSE headers', () => {
    const { subscribe } = progressHandler({ start: jest.fn() });
    const req = makeFakeReq({ id: 'proj-1' });
    const res = makeFakeRes();

    subscribe(req, res);

    expect(res.status).toHaveBeenCalledWith(200);
    expect(res.set).toHaveBeenCalledWith(
      expect.objectContaining({ 'Content-Type': 'text/event-stream' }),
    );
  });

  test('dispatch writes matching messages to subscribed connections as SSE data', () => {
    const { subscribe, dispatch } = progressHandler({ start: jest.fn() });
    const req = makeFakeReq({ id: 'proj-1' });
    const res = makeFakeRes();
    subscribe(req, res);

    dispatch({ project_id: 'proj-1', step: 'render', status: 'in_progress' });

    expect(res.write).toHaveBeenCalledWith(
      `data: ${JSON.stringify({ project_id: 'proj-1', step: 'render', status: 'in_progress' })}\n\n`,
    );
  });

  test('dispatch silently drops messages when no subscriber for project_id', () => {
    const { dispatch } = progressHandler({ start: jest.fn() });
    expect(() => dispatch({ project_id: 'nobody-subscribed' })).not.toThrow();
  });

  test('cleans up connection on req close', () => {
    const { subscribe, dispatch } = progressHandler({ start: jest.fn() });
    const req = makeFakeReq({ id: 'proj-1' });
    const res = makeFakeRes();
    subscribe(req, res);

    req._trigger('close');
    dispatch({ project_id: 'proj-1', step: 'x' });

    expect(res.write).not.toHaveBeenCalled();
  });

  test('rejects invalid project id path param', () => {
    const { subscribe } = progressHandler({ start: jest.fn() });
    const req = makeFakeReq({ id: '../etc/passwd' });
    const res = makeFakeRes();

    subscribe(req, res);

    expect(res.status).toHaveBeenCalledWith(400);
    expect(res.json).toHaveBeenCalledWith({ error: 'invalid_project_id' });
  });
});
