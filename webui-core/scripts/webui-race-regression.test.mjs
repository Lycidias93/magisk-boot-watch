import test from 'node:test';
import assert from 'node:assert/strict';
import fs from 'node:fs';
import vm from 'node:vm';

class HeadersLite {
  constructor(values = {}) { this.values = new Map(Object.entries(values)); }
  get(name) { return this.values.get(name) || this.values.get(String(name).toLowerCase()) || null; }
}

class ResponseLite {
  constructor(body = '', options = {}) {
    this.body = String(body);
    this.status = options.status || 200;
    this.ok = this.status >= 200 && this.status < 300;
    this.statusText = this.ok ? 'OK' : 'Conflict';
    this.headers = new HeadersLite(options.headers || {});
  }
  clone() { return new ResponseLite(this.body, { status: this.status, headers: Object.fromEntries(this.headers.values) }); }
  async json() { return JSON.parse(this.body || '{}'); }
  async text() { return this.body; }
}

function deferred() {
  let resolve;
  let reject;
  const promise = new Promise((res, rej) => { resolve = res; reject = rej; });
  return { promise, resolve, reject };
}

function loadGuard(fetchImpl) {
  const root = { dataset: {} };
  const listeners = {};
  const context = {
    console,
    URL,
    Response: ResponseLite,
    location: { href: 'http://127.0.0.1:20000/' },
    fetch: fetchImpl,
    setTimeout,
    clearTimeout,
    addEventListener() {},
    document: {
      documentElement: root,
      addEventListener(type, handler) { listeners[type] = handler; },
    },
  };
  context.globalThis = context;
  vm.createContext(context);
  vm.runInContext(fs.readFileSync(new URL('../module/webroot/race-guard.js', import.meta.url), 'utf8'), context);
  return { context, root, listeners };
}

const ok = body => new ResponseLite(body, { status: 200, headers: { 'Content-Type': 'application/json' } });

test('mutations stay locked until first successful status response', async () => {
  const calls = [];
  const { context, root } = loadGuard(async (input, options = {}) => {
    calls.push([String(input), options.method || 'GET']);
    return ok('{"ok":true}');
  });
  assert.equal(root.dataset.webuiStatusReady, 'false');
  const blocked = await context.fetch('/api/v1/action', { method: 'POST', body: '{}' });
  assert.equal(blocked.status, 409);
  assert.equal(calls.length, 0);
  await context.fetch('/api/v1/status');
  assert.equal(root.dataset.webuiStatusReady, 'true');
});

test('a second mutation is rejected until the completion status refresh', async () => {
  let postCount = 0;
  const { context, root } = loadGuard(async (input, options = {}) => {
    const path = new URL(String(input), 'http://127.0.0.1/').pathname;
    if ((options.method || 'GET') === 'POST') postCount += 1;
    return ok(JSON.stringify({ ok: true, path }));
  });
  await context.fetch('/api/v1/status');
  const first = await context.fetch('/api/v1/action', { method: 'POST', body: '{}' });
  assert.equal(first.status, 200);
  assert.equal(root.dataset.webuiMutationBusy, 'true');
  const duplicate = await context.fetch('/api/v1/action', { method: 'POST', body: '{}' });
  assert.equal(duplicate.status, 409);
  assert.equal(postCount, 1);
  await context.fetch('/api/v1/status');
  assert.equal(root.dataset.webuiMutationBusy, 'false');
});

test('out-of-order log responses resolve to the latest requested log body', async () => {
  const first = deferred();
  const second = deferred();
  let count = 0;
  const { context } = loadGuard((input) => {
    if (String(input).startsWith('/api/v1/log')) return (++count === 1 ? first : second).promise;
    return Promise.resolve(ok('{"ok":true}'));
  });
  const oldRequest = context.fetch('/api/v1/log?lines=50');
  const newRequest = context.fetch('/api/v1/log?lines=300');
  second.resolve(ok('new-log'));
  await new Promise(resolve => setTimeout(resolve, 0));
  first.resolve(ok('old-log'));
  assert.equal(await (await newRequest).text(), 'new-log');
  assert.equal(await (await oldRequest).text(), 'new-log');
});

test('job launch busy state clears on the next successful jobs refresh', async () => {
  const { context, root } = loadGuard(async () => ok('{"ok":true,"data":[]}'));
  await context.fetch('/api/v1/status');
  await context.fetch('/api/v1/jobs', { method: 'POST', body: '{}' });
  assert.equal(root.dataset.webuiMutationBusy, 'true');
  await context.fetch('/api/v1/jobs');
  assert.equal(root.dataset.webuiMutationBusy, 'false');
});
