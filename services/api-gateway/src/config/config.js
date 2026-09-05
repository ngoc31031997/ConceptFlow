'use strict';

/**
 * Loads Gateway configuration from environment variables.
 *
 * Kept as a single small module (rather than scattering `process.env` reads
 * across the codebase) so every consumer sees the same defaults and so
 * tests can construct a config object without touching real env vars.
 *
 * @returns {{
 *   orchestratorUrl: string,
 *   contentPluginUrl: string,
 *   publisherUrl: string,
 *   rabbitmqUrl: string,
 *   port: number,
 *   webGuiOrigin: string,
 * }}
 */
function loadConfig(env = process.env) {
  return {
    orchestratorUrl: env.ORCHESTRATOR_URL || 'http://orchestrator:8000',
    contentPluginUrl: env.CONTENT_PLUGIN_URL || 'http://content-plugin:8000',
    publisherUrl: env.PUBLISHER_URL || 'http://publisher:8000',
    rabbitmqUrl: env.RABBITMQ_URL || 'amqp://guest:guest@rabbitmq:5672/',
    port: Number(env.PORT) || 8080,
    webGuiOrigin: env.WEB_GUI_ORIGIN || 'http://localhost:3000',
  };
}

module.exports = { loadConfig };
