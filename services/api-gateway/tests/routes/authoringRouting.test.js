'use strict';

const express = require('express');
const request = require('supertest');
const { projectsRouter } = require('../../src/routes/projects');
const { promptsRouter } = require('../../src/routes/prompts');

const ok = () => ({ request: jest.fn().mockResolvedValue({ status: 200, headers: new Map(), body: {} }) });

function buildApp(clients) {
  const app = express();
  app.use(express.json());
  app.use((req, _res, next) => {
    req.requestId = 'test-id';
    next();
  });
  app.use(projectsRouter(clients.orchestrator, '/shared', clients.orchestratorAi, clients.authoring, clients.authoringAi));
  app.use(promptsRouter(clients.authoring));
  return app;
}

// CR-040 FR111: prompts and the 1a/1b/1c chain are authoring-service's; the
// project itself stays with the orchestrator.
describe('routing between orchestrator and authoring-service', () => {
  let clients;
  beforeEach(() => {
    clients = { orchestrator: ok(), orchestratorAi: ok(), authoring: ok(), authoringAi: ok() };
  });

  test.each([
    ['get', '/v1/projects/p1/authoring', 'authoring'],
    ['post', '/v1/projects/p1/authoring/story', 'authoring'],
    ['put', '/v1/projects/p1/authoring/mode', 'authoring'],
    ['get', '/v1/projects/p1/authoring/story/progress', 'authoring'],
    ['post', '/v1/projects/p1/authoring/chain', 'authoring'],
    ['delete', '/v1/projects/p1/authoring/chain', 'authoring'],
    ['post', '/v1/projects/p1/authoring/story/generate', 'authoringAi'],
    ['post', '/v1/projects/p1/suggest-metadata', 'authoringAi'],
    ['post', '/v1/short-script-suggestions', 'authoringAi'],
    ['get', '/v1/prompts/story_architect', 'authoring'],
    ['post', '/v1/prompt-renders', 'authoring'],
    ['get', '/v1/script-templates', 'authoring'],
    ['get', '/v1/admin/prompts', 'authoring'],
    ['get', '/v1/llm/status', 'authoring'],
    ['get', '/v1/operations/op-1', 'authoring'],
    ['get', '/v1/operations/delete:p1', 'orchestrator'],
    ['get', '/v1/projects/p1', 'orchestrator'],
    ['post', '/v1/projects', 'orchestrator'],
    ['patch', '/v1/projects/p1/settings', 'orchestrator'],
    ['get', '/v1/projects/p1/errors', 'orchestrator'],
  ])('%s %s goes to %s', async (method, url, target) => {
    await request(buildApp(clients))[method](url).send({});
    expect(clients[target].request).toHaveBeenCalledTimes(1);
    for (const other of Object.keys(clients).filter((k) => k !== target)) {
      expect(clients[other].request).not.toHaveBeenCalled();
    }
  });
});
