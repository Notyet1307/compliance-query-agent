#!/usr/bin/env node
// Synthetic trials only. Observation configuration is operator-owned, never RPC input.
import { appendFileSync, readFileSync } from 'node:fs';
import { createHash, randomUUID } from 'node:crypto';
import { setTimeout } from 'node:timers/promises';
import { defineService, runServiceMain } from '@chaitin-ai/octobus-sdk';

const corpus = JSON.parse(readFileSync(new URL('../data/demo-corpus.json', import.meta.url)));
const observationRoot = process.env.CQA_OBSERVATION_ROOT || '/var/lib/octobus/x1';

const service = defineService({
  handlers: {
    'compliance.v1.KnowledgeService/Search': async ({ request }) => {
      if (request?.schemaVersion !== 'cqa.search/v1' || typeof request.query !== 'string' ||
          !Number.isInteger(request.limit) || request.limit < 1 || request.limit > 100) {
        throw new Error('X1_REQUEST_INVALID');
      }
      if (corpus.synthetic !== true || corpus.sources.some(source => source.synthetic !== true)) {
        throw new Error('X1_SYNTHETIC_ONLY');
      }
      const { delayMs } = JSON.parse(readFileSync(`${observationRoot}/control.json`, 'utf8'));
      if (!Number.isInteger(delayMs) || delayMs < 0 || delayMs > 20000) {
        throw new Error('X1_CONTROL_INVALID');
      }
      const invocation = randomUUID();
      const requestDigest = createHash('sha256').update(JSON.stringify(request)).digest('hex');
      const observe = event => appendFileSync(`${observationRoot}/events.ndjson`,
        JSON.stringify({ event, invocation, requestDigest, at: new Date().toISOString() }) + '\n', { mode: 0o600 });
      observe('entered');
      if (delayMs) await setTimeout(delayMs);
      const result = {
        corpus: { ...corpus, sources: corpus.sources.slice(0, request.limit) },
        truncated: corpus.sources.length > request.limit,
      };
      observe('completed');
      return result;
    },
  },
});

runServiceMain(service);
