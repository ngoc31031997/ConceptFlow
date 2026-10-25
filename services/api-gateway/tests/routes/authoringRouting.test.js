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

// Prompts and the 1a/1b/1c chain are authoring-service's; the
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
    ['get', '/v1/projects/p1/authoring/code/segments', 'authoring'],
    ['get', '/v1/projects/p1/authoring/code/segments/1.1-1.3/prompt', 'authoring'],
    ['put', '/v1/projects/p1/authoring/code/segments/1.1-1.3', 'authoring'],
    ['put', '/v1/projects/p1/authoring/code/chunk-shots', 'authoring'],
    ['get', '/v1/projects/p1/authoring/story/progress', 'authoring'],
    ['post', '/v1/projects/p1/authoring/chain', 'authoring'],
    ['delete', '/v1/projects/p1/authoring/chain', 'authoring'],
    ['post', '/v1/projects/p1/authoring/story/generate', 'authoringAi'],
    ['post', '/v1/projects/p1/suggest-metadata', 'authoringAi'],
    ['get', '/v1/projects/p1/idea', 'authoring'],
    ['post', '/v1/projects/p1/idea/generate', 'authoringAi'],
    ['put', '/v1/projects/p1/idea', 'authoring'],
    ['post', '/v1/projects/p1/idea/approve', 'authoring'],
    ['post', '/v1/projects/p1/idea/unapprove', 'authoring'],
    ['post', '/v1/projects/p1/idea/review', 'authoringAi'],
    ['get', '/v1/projects/p1/creative-brief', 'authoring'],
    ['post', '/v1/projects/p1/creative-brief/generate', 'authoringAi'],
    ['put', '/v1/projects/p1/creative-brief', 'authoring'],
    ['post', '/v1/projects/p1/creative-brief/approve', 'authoring'],
    ['post', '/v1/projects/p1/creative-brief/unapprove', 'authoring'],
    ['post', '/v1/projects/p1/creative-brief/review', 'authoringAi'],
    ['post', '/v1/projects/p1/units/idea/main/redo', 'authoringAi'],
    ['get', '/v1/projects/p1/brief', 'authoring'],
    ['put', '/v1/projects/p1/brief/target-duration', 'authoring'],
    ['put', '/v1/projects/p1/brief/platform', 'authoring'],
    ['post', '/v1/projects/p1/brief/generate', 'authoringAi'],
    ['get', '/v1/projects/p1/units/brief/main', 'authoring'],
    ['put', '/v1/projects/p1/units/brief/main', 'authoring'],
    ['post', '/v1/projects/p1/units/brief/main/redo', 'authoringAi'],
    ['post', '/v1/projects/p1/units/brief/main/approve', 'authoring'],
    ['post', '/v1/projects/p1/units/brief/main/unapprove', 'authoring'],
    ['get', '/v1/projects/p1/units/brief/main/impact', 'authoring'],
    ['get', '/v1/projects/p1/research', 'authoring'],
    ['post', '/v1/projects/p1/research/questions/generate', 'authoringAi'],
    ['post', '/v1/projects/p1/research/questions', 'authoring'],
    ['delete', '/v1/projects/p1/research/questions/Q2', 'authoring'],
    ['get', '/v1/projects/p1/research/prompt', 'authoring'],
    ['get', '/v1/projects/p1/research/follow-up-prompt', 'authoring'],
    ['post', '/v1/projects/p1/research/paste', 'authoring'],
    ['post', '/v1/projects/p1/research/facts', 'authoring'],
    ['put', '/v1/projects/p1/research/facts/F01', 'authoring'],
    ['post', '/v1/projects/p1/research/verify', 'authoring'],
    ['put', '/v1/projects/p1/research/skip', 'authoring'],
    ['get', '/v1/projects/p1/story', 'authoring'],
    ['post', '/v1/projects/p1/story/generate', 'authoringAi'],
    ['put', '/v1/projects/p1/story', 'authoring'],
    ['post', '/v1/projects/p1/story/approve', 'authoring'],
    ['post', '/v1/projects/p1/story/unapprove', 'authoring'],
    ['get', '/v1/projects/p1/beats', 'authoring'],
    ['post', '/v1/projects/p1/beats/skeleton', 'authoring'],
    ['post', '/v1/projects/p1/beats/skeleton/approve', 'authoring'],
    ['post', '/v1/projects/p1/beats/generate', 'authoringAi'],
    ['post', '/v1/projects/p1/beats', 'authoringAi'],
    ['put', '/v1/projects/p1/beats/order', 'authoring'],
    ['delete', '/v1/projects/p1/beats/b2', 'authoring'],
    ['get', '/v1/projects/p1/script', 'authoring'],
    ['post', '/v1/projects/p1/script/generate', 'authoringAi'],
    ['post', '/v1/projects/p1/script/suggest', 'authoringAi'],
    ['post', '/v1/projects/p1/script/speak', 'authoring'],
    ['get', '/v1/projects/p1/script-lock', 'authoring'],
    ['post', '/v1/projects/p1/script-lock/review', 'authoringAi'],
    ['post', '/v1/projects/p1/script-lock/findings/r1/apply', 'authoring'],
    ['post', '/v1/projects/p1/script-lock/findings/m-must/accept', 'authoring'],
    ['post', '/v1/projects/p1/script-lock/notes', 'authoring'],
    ['post', '/v1/projects/p1/script-lock/glossary', 'authoring'],
    ['post', '/v1/projects/p1/script-lock/handoff/generate', 'authoringAi'],
    ['post', '/v1/projects/p1/script-lock/handoff/b2/confirm', 'authoring'],
    ['post', '/v1/projects/p1/script-lock/lock', 'authoring'],
    ['put', '/v1/projects/p1/units/handoff_beat/b2', 'authoring'],
    ['post', '/v1/projects/p1/units/handoff_beat/b2/redo', 'authoringAi'],
    ['put', '/v1/projects/p1/units/script_beat/b2', 'authoring'],
    ['post', '/v1/projects/p1/units/script_beat/b2/redo', 'authoringAi'],
    ['put', '/v1/projects/p1/beats/b2', 'authoring'],
    ['post', '/v1/projects/p1/beats/b2/redo', 'authoringAi'],
    ['post', '/v1/projects/p1/beats/b2/approve', 'authoring'],
    ['post', '/v1/projects/p1/beats/b2/unapprove', 'authoring'],
    ['get', '/v1/projects/p1/beats/b2/impact', 'authoring'],
    ['get', '/v1/projects/p1/units/fact/F01/impact', 'authoring'],
    ['post', '/v1/projects/p1/units/research/main/approve', 'authoring'],
    ['get', '/v1/prompts/story_architect', 'authoring'],
    ['post', '/v1/prompt-renders', 'authoring'],
    ['get', '/v1/admin/prompts', 'authoring'],
    ['get', '/v1/llm/status', 'authoring'],
    ['get', '/v1/settings/ai-model', 'authoring'],
    ['put', '/v1/settings/ai-model/steps/brief', 'authoring'],
    ['get', '/v1/operations/op-1', 'authoring'],
    ['get', '/v1/operations/delete:p1', 'orchestrator'],
    ['post', '/v1/operations/op-1/cancel', 'authoring'],
    ['get', '/v1/projects/p1/operations', 'authoring'],
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
