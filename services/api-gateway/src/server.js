'use strict';

const express = require('express');
const pino = require('pino');

const { loadConfig } = require('./config/config');
const { correlationMiddleware } = require('./middleware/correlation');
const { corsMiddleware } = require('./middleware/cors');
const { createHttpClient } = require('./clients/httpClient');
const { createAmqpClient } = require('./clients/amqpClient');
const { progressHandler } = require('./handlers/progressHandler');
const { pluginsRouter } = require('./routes/plugins');
const { voicesRouter } = require('./routes/voices');
const { sagasRouter } = require('./routes/sagas');
const { projectsRouter } = require('./routes/projects');
const { authRouter } = require('./routes/auth');
const { progressRouter } = require('./routes/progress');
const { healthRouter } = require('./routes/health');

const logger = pino({ level: 'warn' });

/**
 * Composition root — wires config, clients, Express app, and the AMQP
 * consumer together, following the 6-step order in dependency-injection.md.
 * Only `httpClient` (×3) and `amqpClient` are injected via factory params;
 * Express `app`/router objects are constructed directly (they're framework
 * infrastructure, not business abstractions that need independent testing).
 */
function main() {
  // 1. Load config from env vars.
  const config = loadConfig();

  // 2. Initialize httpClient for the 3 targets, plus a longer-timeout
  // variant of the orchestrator client for the AI metadata-suggestion route
  // (local LLM generation can take up to ~2 minutes).
  const orchestratorClient = createHttpClient(config.orchestratorUrl);
  const orchestratorAiClient = createHttpClient(config.orchestratorUrl, { timeoutMs: 130_000 });
  const contentPluginClient = createHttpClient(config.contentPluginUrl);
  const publisherClient = createHttpClient(config.publisherUrl);

  // 3. Connect to RabbitMQ (amqpClient), declare exclusive queue bound to progress.fanout.
  const progress = progressHandler();
  const amqpClient = createAmqpClient(config.rabbitmqUrl, progress.dispatch, { logger });
  amqpClient.start();

  // 4. Initialize Express app, register correlation middleware.
  const app = express();
  app.use(corsMiddleware(config.webGuiOrigin));
  app.use(express.json());
  app.use(correlationMiddleware());

  // 5. Register routes.
  app.use(pluginsRouter(contentPluginClient));
  app.use(voicesRouter(config.sharedDir));
  app.use(sagasRouter(orchestratorClient));
  app.use(projectsRouter(orchestratorClient, config.sharedDir, orchestratorAiClient));
  app.use(authRouter(publisherClient));
  app.use(progressRouter(progress));
  app.use(healthRouter());

  // 6. Start HTTP server (AMQP consumer loop was already started in step 3).
  app.listen(config.port, () => {
    logger.info({ port: config.port }, 'API Gateway listening');
  });

  return app;
}

if (require.main === module) {
  main();
}

module.exports = { main };
