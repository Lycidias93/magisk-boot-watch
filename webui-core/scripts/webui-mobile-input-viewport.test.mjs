import test from 'node:test';
import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';

const index = await readFile(new URL('../module/webroot/index.html', import.meta.url), 'utf8');
const viewportScript = await readFile(new URL('../module/webroot/mobile-input-viewport.js', import.meta.url), 'utf8');

test('mobile input viewport helper is shipped and loaded before app.js', () => {
  const helper = index.indexOf('<script src="mobile-input-viewport.js"></script>');
  const app = index.indexOf('<script src="app.js"></script>');
  assert.ok(helper >= 0);
  assert.ok(app > helper);
});

test('mobile input viewport helper tracks focused controls across keyboard resize', () => {
  assert.match(viewportScript, /window\.visualViewport/);
  assert.match(viewportScript, /focusin/);
  assert.match(viewportScript, /scrollIntoView/);
  assert.match(viewportScript, /viewport\.addEventListener\('resize'/);
  assert.match(viewportScript, /viewport\.addEventListener\('scroll'/);
});
